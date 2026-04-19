package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

// GormTenantStatsRepo implementa application.TenantStatsProvider via GORM/MySQL.
type GormTenantStatsRepo struct{ db *gorm.DB }

func NewGormTenantStatsRepo(db *gorm.DB) *GormTenantStatsRepo {
	return &GormTenantStatsRepo{db: db}
}

// Compile-time check: GormTenantStatsRepo implements the application provider.
var _ application.TenantStatsProvider = (*GormTenantStatsRepo)(nil)

// applyLeadUserScope adiciona o filtro de responsible_user_id quando userID != nil.
// Usado pelos blocos do dashboard tenant para escopar resultados ao usuário comum.
// O argumento leadAlias permite usar o helper com queries que aliasam a tabela leads
// (ex: "l") ou que usam o nome direto (passar "leads").
func applyLeadUserScope(q *gorm.DB, userID *string, leadAlias string) *gorm.DB {
	if userID == nil {
		return q
	}
	return q.Where(leadAlias+".responsible_user_id = ?", *userID)
}

// FunilBlock — Bloco 1: status totals + colunas + conversão + novos hoje/semana.
// Filtro opcional por responsible_user_id quando userID != nil.
// Retorna também o nome do funil ativo (default do tenant).
func (r *GormTenantStatsRepo) FunilBlock(
	ctx context.Context,
	tenantID string,
	userID *string,
	now time.Time,
) (*domain.FunilBlock, string, error) {
	// 1) pegar funil default do tenant
	var f struct {
		ID   string
		Name string
	}
	if err := r.db.WithContext(ctx).Table("funnels").
		Select("id, name").
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		Take(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &domain.FunilBlock{}, "", nil
		}
		return nil, "", err
	}
	funnelID, funnelName := f.ID, f.Name

	// 2) contadores por status
	type sRow struct {
		Status string
		Total  int64
	}
	var srows []sRow
	qStatus := r.db.WithContext(ctx).Table("leads").Where("tenant_id = ?", tenantID)
	if userID != nil {
		qStatus = qStatus.Where("responsible_user_id = ?", *userID)
	}
	if err := qStatus.Select("status, COUNT(*) AS total").Group("status").Scan(&srows).Error; err != nil {
		return nil, "", err
	}
	block := &domain.FunilBlock{}
	for _, rr := range srows {
		switch rr.Status {
		case "open":
			block.StatusTotals.Open = rr.Total
		case "won":
			block.StatusTotals.Won = rr.Total
		case "lost":
			block.StatusTotals.Lost = rr.Total
		}
	}
	totalFin := block.StatusTotals.Won + block.StatusTotals.Lost
	if totalFin > 0 {
		block.ConversionPct = float64(block.StatusTotals.Won) * 100 / float64(totalFin)
	}

	// 3) leads por coluna do funil default — JOIN funnel_columns (NÃO 'columns')
	type cRow struct {
		ColumnID   string
		Name       string
		OrderIndex int
		Total      int64
	}
	var crows []cRow
	qCol := r.db.WithContext(ctx).Table("leads AS l").
		Select("c.id AS column_id, c.name AS name, c.order_index AS order_index, COUNT(l.id) AS total").
		Joins("JOIN funnel_columns c ON c.id = l.column_id").
		Where("l.tenant_id = ? AND l.funnel_id = ?", tenantID, funnelID)
	if userID != nil {
		qCol = qCol.Where("l.responsible_user_id = ?", *userID)
	}
	if err := qCol.Group("c.id, c.name, c.order_index").Order("c.order_index").Scan(&crows).Error; err != nil {
		return nil, "", err
	}
	for _, rr := range crows {
		block.ColumnTotals = append(block.ColumnTotals, domain.ColumnLeadsCount{
			ColumnID:   rr.ColumnID,
			ColumnName: rr.Name,
			OrderIndex: rr.OrderIndex,
			Count:      rr.Total,
		})
	}

	// 4) novos hoje / novos na semana (semana iniciando no domingo, alinhada com time.Weekday)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	qTime := r.db.WithContext(ctx).Table("leads").Where("tenant_id = ?", tenantID)
	if userID != nil {
		qTime = qTime.Where("responsible_user_id = ?", *userID)
	}
	if err := qTime.Session(&gorm.Session{}).Where("created_at >= ?", today).Count(&block.NewToday).Error; err != nil {
		return nil, "", err
	}
	if err := qTime.Session(&gorm.Session{}).Where("created_at >= ?", weekStart).Count(&block.NewThisWeek).Error; err != nil {
		return nil, "", err
	}

	return block, funnelName, nil
}

// ProdutosBlock — Bloco 5: leads agrupados por produto, com totais won/lost.
func (r *GormTenantStatsRepo) ProdutosBlock(
	ctx context.Context,
	tenantID string,
	userID *string,
) ([]domain.ProductLeadsCount, error) {
	type pRow struct {
		ProductID string
		Name      string
		Total     int64
		Won       int64
		Lost      int64
	}
	var rows []pRow
	q := r.db.WithContext(ctx).Table("leads AS l").
		Select(`p.id AS product_id, p.name AS name,
				COUNT(l.id) AS total,
				SUM(CASE WHEN l.status='won' THEN 1 ELSE 0 END) AS won,
				SUM(CASE WHEN l.status='lost' THEN 1 ELSE 0 END) AS lost`).
		// Leads sem product_id são excluídos do bloco por construção (INNER JOIN).
		Joins("JOIN products p ON p.id = l.product_id").
		Where("l.tenant_id = ?", tenantID)
	if userID != nil {
		q = q.Where("l.responsible_user_id = ?", *userID)
	}
	if err := q.Group("p.id, p.name").Order("total DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ProductLeadsCount, 0, len(rows))
	for _, rr := range rows {
		out = append(out, domain.ProductLeadsCount{
			ProductID:   rr.ProductID,
			ProductName: rr.Name,
			Total:       rr.Total,
			Won:         rr.Won,
			Lost:        rr.Lost,
		})
	}
	return out, nil
}

// WhatsAppBlock — Bloco 2: agregados de conversas/mensagens do tenant.
//   - ActiveConversations: COUNT(DISTINCT c.id) onde c.status='open'.
//   - IncomingMessages / OutgoingMessages: contagem de messages por direction
//     juntando em conversations do tenant.
//   - FirstResponseAvgSec: tempo médio (s) entre primeira incoming e primeira
//     outgoing por conversa, considerando apenas conversas com first_outgoing
//     >= first_incoming. COALESCE(..., 0) garante zero quando não há dados.
//
// Quando userID != nil, restringe via leads.conversation_id = c.id AND
// leads.responsible_user_id = userID em todas as agregações.
func (r *GormTenantStatsRepo) WhatsAppBlock(ctx context.Context, tenantID string, userID *string) (*domain.WhatsAppStats, error) {
	out := &domain.WhatsAppStats{}

	// Conversas ativas (status='open')
	convQ := r.db.WithContext(ctx).Table("conversations AS c").
		Where("c.tenant_id = ? AND c.status = ?", tenantID, "open")
	if userID != nil {
		convQ = convQ.Joins("JOIN leads l ON l.conversation_id = c.id").
			Where("l.responsible_user_id = ?", *userID)
	}
	var active int64
	if err := convQ.Distinct("c.id").Count(&active).Error; err != nil {
		return nil, err
	}
	out.ActiveConversations = active

	// Mensagens incoming/outgoing
	type mRow struct {
		Direction string
		Total     int64
	}
	var mrows []mRow
	msgQ := r.db.WithContext(ctx).Table("messages AS m").
		Select("m.direction AS direction, COUNT(*) AS total").
		Joins("JOIN conversations c ON c.id = m.conversation_id").
		Where("c.tenant_id = ?", tenantID)
	if userID != nil {
		msgQ = msgQ.Joins("JOIN leads l ON l.conversation_id = c.id").
			Where("l.responsible_user_id = ?", *userID)
	}
	if err := msgQ.Group("m.direction").Scan(&mrows).Error; err != nil {
		return nil, err
	}
	for _, rr := range mrows {
		switch rr.Direction {
		case "incoming":
			out.IncomingMessages = rr.Total
		case "outgoing":
			out.OutgoingMessages = rr.Total
		}
	}

	// Tempo médio de primeira resposta (segundos). Conversas com both incoming e outgoing
	// onde first_outgoing.timestamp >= first_incoming.timestamp.
	type rtRow struct{ DiffSec float64 }
	sqlRaw := `
	SELECT COALESCE(AVG(TIMESTAMPDIFF(SECOND, first_in.t, first_out.t)), 0) AS diff_sec
	FROM (
		SELECT m.conversation_id AS cid, MIN(m.timestamp) AS t
		FROM messages m JOIN conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = ? AND m.direction = 'incoming'
		GROUP BY m.conversation_id
	) first_in
	JOIN (
		SELECT m.conversation_id AS cid, MIN(m.timestamp) AS t
		FROM messages m JOIN conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = ? AND m.direction = 'outgoing'
		GROUP BY m.conversation_id
	) first_out ON first_out.cid = first_in.cid
	WHERE first_out.t >= first_in.t`
	args := []any{tenantID, tenantID}
	if userID != nil {
		sqlRaw += ` AND first_in.cid IN (SELECT conversation_id FROM leads WHERE tenant_id = ? AND responsible_user_id = ?)`
		args = append(args, tenantID, *userID)
	}
	var rr rtRow
	if err := r.db.WithContext(ctx).Raw(sqlRaw, args...).Scan(&rr).Error; err != nil {
		return nil, err
	}
	out.FirstResponseAvgSec = rr.DiffSec
	return out, nil
}

// ResponsaveisBlock — Bloco 3: leads agrupados por responsible_user_id (LEFT JOIN users
// para obter o nome). Leads sem responsável são excluídos. Ordenado por total DESC.
// Filtro opcional por responsible_user_id quando userID != nil (resulta em até 1 linha).
func (r *GormTenantStatsRepo) ResponsaveisBlock(ctx context.Context, tenantID string, userID *string) ([]domain.ResponsiblePerformance, error) {
	type row struct {
		UserID string
		Name   string
		Total  int64
		Won    int64
		Lost   int64
	}
	var rows []row
	q := r.db.WithContext(ctx).Table("leads AS l").
		Select(`l.responsible_user_id AS user_id, u.name AS name,
				COUNT(l.id) AS total,
				SUM(CASE WHEN l.status='won' THEN 1 ELSE 0 END) AS won,
				SUM(CASE WHEN l.status='lost' THEN 1 ELSE 0 END) AS lost`).
		// LEFT JOIN preserva leads cujo responsible_user_id aponta para usuário deletado/inexistente
		// (não há FK enforçando users.id). Esses casos aparecem com UserName vazio, em vez de sumirem do relatório.
		Joins("LEFT JOIN users u ON u.id = l.responsible_user_id").
		Where("l.tenant_id = ? AND l.responsible_user_id IS NOT NULL", tenantID)
	q = applyLeadUserScope(q, userID, "l")
	if err := q.Group("l.responsible_user_id, u.name").Order("total DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ResponsiblePerformance, 0, len(rows))
	for _, rr := range rows {
		out = append(out, domain.ResponsiblePerformance{
			UserID:   rr.UserID,
			UserName: rr.Name,
			Total:    rr.Total,
			Won:      rr.Won,
			Lost:     rr.Lost,
		})
	}
	return out, nil
}

// TempoFunilBlock — Bloco 4: tempo médio (em horas) que cada lead permanece na coluna atual
// e quantos estão "parados" há mais de 7 dias (status='open'). Limita-se ao funil default
// do tenant; quando o tenant não possui funil default, retorna (nil, nil).
// Filtro opcional por responsible_user_id quando userID != nil.
func (r *GormTenantStatsRepo) TempoFunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) ([]domain.ColumnDwell, error) {
	// Funil default
	var f struct{ ID string }
	if err := r.db.WithContext(ctx).Table("funnels").
		Select("id").
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		Take(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	funnelID := f.ID

	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	type row struct {
		ColumnID   string
		Name       string
		OrderIndex int
		AvgHours   float64
		Stuck      int64
	}
	var rows []row
	// AvgHours considera todos os leads (incluindo won/lost) — representa tempo histórico de processamento.
	// StuckOver7Days conta apenas status='open' — representa trabalho pendente.
	q := r.db.WithContext(ctx).Table("leads AS l").
		Select(`c.id AS column_id, c.name AS name, c.order_index AS order_index,
				AVG(TIMESTAMPDIFF(HOUR, l.column_entered_at, ?)) AS avg_hours,
				SUM(CASE WHEN l.column_entered_at < ? AND l.status='open' THEN 1 ELSE 0 END) AS stuck`, now, sevenDaysAgo).
		Joins("JOIN funnel_columns c ON c.id = l.column_id").
		Where("l.tenant_id = ? AND l.funnel_id = ?", tenantID, funnelID)
	q = applyLeadUserScope(q, userID, "l")
	if err := q.Group("c.id, c.name, c.order_index").Order("c.order_index").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.ColumnDwell, 0, len(rows))
	for _, rr := range rows {
		out = append(out, domain.ColumnDwell{
			ColumnID:       rr.ColumnID,
			ColumnName:     rr.Name,
			OrderIndex:     rr.OrderIndex,
			AvgHours:       rr.AvgHours,
			StuckOver7Days: rr.Stuck,
		})
	}
	return out, nil
}
