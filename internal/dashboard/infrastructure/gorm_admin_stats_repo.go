package infrastructure

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
	pagamentosDomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

// GormAdminStatsRepo implementa application.AdminStatsProvider via GORM/MySQL.
// Cobre os blocos 1 (tenants), 2 (uso), 3 (health), 5 (especialistas) e 6 (financeiro)
// do dashboard admin. O bloco 6 delega o cálculo de receita/pendente/atrasado/top-overdue
// para o PaymentRepository (interface do módulo pagamentos), mantendo a SQL financeira
// em um único lugar; a distribuição de planos é consultada direto em tenants.plano.
type GormAdminStatsRepo struct {
	db       *gorm.DB
	payments pagamentosDomain.PaymentRepository
}

func NewGormAdminStatsRepo(db *gorm.DB, payments pagamentosDomain.PaymentRepository) *GormAdminStatsRepo {
	return &GormAdminStatsRepo{db: db, payments: payments}
}

// Compile-time check: GormAdminStatsRepo implements the application provider.
var _ application.AdminStatsProvider = (*GormAdminStatsRepo)(nil)

// TenantsBlock — Bloco 1: totais por status, novos no m\u00eas e s\u00e9rie dos \u00faltimos 6 meses.
// A janela de 6 meses inclui o m\u00eas atual; meses sem registros recebem Count=0 (backfill).
func (r *GormAdminStatsRepo) TenantsBlock(ctx context.Context, now time.Time) (*domain.TenantsBlock, error) {
	out := &domain.TenantsBlock{}

	// 1) totais por status
	type sRow struct {
		Status string
		Total  int64
	}
	var srows []sRow
	if err := r.db.WithContext(ctx).Table("tenants").
		Select("status, COUNT(*) AS total").
		Group("status").Scan(&srows).Error; err != nil {
		return nil, err
	}
	for _, rr := range srows {
		switch rr.Status {
		case "active":
			out.Totals.Active = rr.Total
		case "inactive":
			out.Totals.Inactive = rr.Total
		case "blocked":
			out.Totals.Blocked = rr.Total
		}
	}

	// 2) novos este m\u00eas
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if err := r.db.WithContext(ctx).Table("tenants").
		Where("created_at >= ?", monthStart).
		Count(&out.NewThisMonth).Error; err != nil {
		return nil, err
	}

	// 3) \u00faltimos 6 meses (incluindo o atual). Backfill com 0 para meses sem registros.
	// Janela: primeiro dia de (m\u00eas atual - 5).
	sixMonthsAgo := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, now.Location())
	type mRow struct {
		Y     int
		M     int
		Total int64
	}
	var mrows []mRow
	if err := r.db.WithContext(ctx).Table("tenants").
		Select("YEAR(created_at) AS y, MONTH(created_at) AS m, COUNT(*) AS total").
		Where("created_at >= ?", sixMonthsAgo).
		Group("y, m").Scan(&mrows).Error; err != nil {
		return nil, err
	}
	counts := make(map[string]int64, len(mrows))
	for _, rr := range mrows {
		counts[fmt.Sprintf("%04d-%02d", rr.Y, rr.M)] = rr.Total
	}
	out.Last6Months = make([]domain.MonthlyCount, 0, 6)
	for i := 5; i >= 0; i-- {
		// time.Date normaliza overflow/underflow no Month, ajustando Year.
		m := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
		label := m.Format("2006-01")
		out.Last6Months = append(out.Last6Months, domain.MonthlyCount{
			Label: label,
			Count: counts[label], // 0 se n\u00e3o houver linha
		})
	}

	return out, nil
}

// UsageBlock — Bloco 2: contadores globais (todos tenants).
//   - TotalLeads: COUNT(leads).
//   - TotalMessages: COUNT(messages).
//   - ActiveConversations: COUNT(conversations) onde status='open'.
func (r *GormAdminStatsRepo) UsageBlock(ctx context.Context) (*domain.UsageBlock, error) {
	out := &domain.UsageBlock{}
	if err := r.db.WithContext(ctx).Table("leads").Count(&out.TotalLeads).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Table("messages").Count(&out.TotalMessages).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Table("conversations").
		Where("status = ?", "open").Count(&out.ActiveConversations).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// HealthBlock — Bloco 3: top 10 tenants por COUNT(leads) + lista de inativos h\u00e1 +30d.
//
// last_activity = GREATEST de tr\u00eas ramos, cada um com COALESCE(MAX(...), t.created_at):
// MAX(lead.created_at), MAX(message.timestamp) e t.created_at — quando o ramo n\u00e3o tem
// linhas para o tenant, cai em t.created_at, e o GREATEST elege o instante mais recente
// entre os tr\u00eas. CAST(... AS DATETIME) externo \u00e9 workaround do driver MySQL para a coluna
// ser scaneada como time.Time em vez de []uint8. DaysInactive \u00e9 calculado em Go via
// now.Sub(last_activity).Hours()/24 (truncamento inteiro; testes usam InDelta(2) para
// absorver efeitos de hora-fronteira). LIMIT 50 \u00e9 defensivo.
func (r *GormAdminStatsRepo) HealthBlock(ctx context.Context, now time.Time) (*domain.HealthBlock, error) {
	out := &domain.HealthBlock{}

	// Top 10 tenants por COUNT(leads), ordenado desc.
	type topRow struct {
		TenantID   string
		TenantName string
		Total      int64
	}
	var tops []topRow
	// "Active" aqui refere-se a volume de leads (uso histórico), não ao status operacional do tenant.
	// Tenants com status='blocked'/'inactive' aparecem se tiveram leads — admin precisa ver uso real.
	// Se o template precisar destacar visualmente, fazer no template (não filtrar aqui).
	if err := r.db.WithContext(ctx).
		Table("leads AS l").
		Select("l.tenant_id AS tenant_id, t.name AS tenant_name, COUNT(*) AS total").
		Joins("JOIN tenants t ON t.id = l.tenant_id").
		Group("l.tenant_id, t.name").
		Order("total DESC").
		Limit(10).Scan(&tops).Error; err != nil {
		return nil, err
	}
	for _, tr := range tops {
		out.Top10Active = append(out.Top10Active, domain.TenantActivity{
			TenantID:   tr.TenantID,
			TenantName: tr.TenantName,
			LeadCount:  tr.Total,
		})
	}

	// Inativos: tenants cujo last_activity < now - 30d.
	cutoff := now.AddDate(0, 0, -30)
	type inactiveRow struct {
		ID           string
		Name         string
		LastActivity time.Time
	}
	var inactiveRows []inactiveRow
	// last_activity = GREATEST de:
	//   - max lead.created_at (ou t.created_at, se não houver leads),
	//   - max message.timestamp (ou t.created_at, se não houver mensagens),
	//   - t.created_at.
	// Cada COALESCE garante que ramos sem atividade caiam para t.created_at; o GREATEST
	// elege o instante mais recente entre as três fontes. CAST AS DATETIME no resultado
	// final fixa o tipo da coluna para o driver MySQL → time.Time (sem ambiguidade de
	// VARCHAR causada por inferência de COALESCE/GREATEST).
	sqlRaw := `
	SELECT t.id AS id, t.name AS name,
		CAST(GREATEST(
			COALESCE((SELECT MAX(l.created_at) FROM leads l WHERE l.tenant_id = t.id), t.created_at),
			COALESCE((SELECT MAX(m.timestamp) FROM messages m JOIN conversations c ON c.id = m.conversation_id WHERE c.tenant_id = t.id), t.created_at),
			t.created_at
		) AS DATETIME) AS last_activity
	FROM tenants t
	HAVING last_activity < ?
	ORDER BY last_activity ASC
	LIMIT 50`
	if err := r.db.WithContext(ctx).Raw(sqlRaw, cutoff).Scan(&inactiveRows).Error; err != nil {
		return nil, err
	}
	for _, rr := range inactiveRows {
		days := int64(now.Sub(rr.LastActivity).Hours() / 24)
		out.InactiveList = append(out.InactiveList, domain.InactiveTenant{
			TenantID:     rr.ID,
			TenantName:   rr.Name,
			DaysInactive: days,
		})
	}

	return out, nil
}

// EspecialistasBlock — Bloco 5: total de especialistas, contador de qualificações
// e distribuição por tenant via junction specialist_tenants.
//
// Qualifications fica em 0 enquanto a F05 não expõe um contador de qualificações
// (não há tabela `specialist_qualifications` hoje). Quando o módulo de qualificações
// publicar a contagem (via repo/serviço dedicado), substituir aqui.
func (r *GormAdminStatsRepo) EspecialistasBlock(ctx context.Context) (*domain.SpecialistsBlock, error) {
	out := &domain.SpecialistsBlock{}

	// Total de especialistas (todos, independente de status).
	if err := r.db.WithContext(ctx).Table("specialists").Count(&out.Total).Error; err != nil {
		return nil, err
	}

	// TODO(F05): popular Qualifications quando o módulo de qualificações
	// expuser o contador (tabela/serviço ainda não disponíveis).
	out.Qualifications = 0

	// Distribuição por tenant via junction table specialist_tenants.
	type byTenantRow struct {
		TenantID   string
		TenantName string
		Total      int64
	}
	var rows []byTenantRow
	if err := r.db.WithContext(ctx).
		Table("specialist_tenants AS st").
		Select("st.tenant_id AS tenant_id, t.name AS tenant_name, COUNT(*) AS total").
		Joins("JOIN tenants t ON t.id = st.tenant_id").
		Group("st.tenant_id, t.name").
		Order("total DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, rr := range rows {
		out.ByTenant = append(out.ByTenant, domain.SpecialistByTenant{
			TenantID:   rr.TenantID,
			TenantName: rr.TenantName,
			Total:      rr.Total,
		})
	}
	return out, nil
}

// FinanceiroBlock — Bloco 6: receita do ano, pendente/atrasado totais, distribuição
// de planos e top-10 tenants em atraso.
//
// Receita/pendente/atrasado/top-overdue vêm de PaymentRepository.GlobalSummary —
// reusamos a SQL canônica do módulo pagamentos (single source of truth para regras
// financeiras). PlanDistribution é uma agregação simples sobre tenants.plano e fica
// neste repo para evitar acoplar pagamentos a conhecimento de planos por contagem.
//
// Plans desconhecidos (string fora do conjunto suportado: mensal/anual/vitalicio/externo)
// são silenciosamente ignorados — o admin vê apenas categorias conhecidas; isso evita
// quebrar o dashboard caso um valor exótico apareça (ex.: migração futura).
func (r *GormAdminStatsRepo) FinanceiroBlock(ctx context.Context, now time.Time) (*domain.FinancialBlock, error) {
	gs, err := r.payments.GlobalSummary(ctx, now)
	if err != nil {
		return nil, err
	}

	type pRow struct {
		Plano string
		Total int64
	}
	var prows []pRow
	if err := r.db.WithContext(ctx).Table("tenants").
		Select("plano, COUNT(*) AS total").
		Where("plano IS NOT NULL AND plano <> ''").
		Group("plano").Scan(&prows).Error; err != nil {
		return nil, err
	}
	var dist domain.PlanDistribution
	for _, rr := range prows {
		switch rr.Plano {
		case "mensal":
			dist.Mensal = rr.Total
		case "anual":
			dist.Anual = rr.Total
		case "vitalicio":
			dist.Vitalicio = rr.Total
		case "externo":
			dist.Externo = rr.Total
		}
	}

	out := &domain.FinancialBlock{
		ReceitaAnoCents:    gs.TotalPagoAnoCents,
		PendenteTotalCents: gs.TotalPendenteCents,
		AtrasadoTotalCents: gs.TotalAtrasadoCents,
		PlanDist:           dist,
	}
	for _, t := range gs.TopOverdue {
		out.TopOverdue = append(out.TopOverdue, domain.OverdueTenant{
			TenantID:   t.TenantID,
			TenantName: t.TenantName,
			ValorCents: t.ValorCents,
		})
	}
	return out, nil
}
