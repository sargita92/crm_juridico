package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	pagamentosInfra "github.com/sasrgita/crm-juridico/internal/pagamentos/infrastructure"
)

// setupAdminRepo reaproveita o cleanup/migrations de setupStatsRepo e devolve o
// repositório admin sobre o mesmo *gorm.DB já limpo. Constrói também um
// PaymentRepository real sobre o mesmo *gorm.DB para que FinanceiroBlock exercite
// o SQL de GlobalSummary contra a base do testcontainer (integração real).
func setupAdminRepo(t *testing.T) (*GormAdminStatsRepo, *gorm.DB) {
	t.Helper()
	_, db := setupStatsRepo(t)
	payments := pagamentosInfra.NewGormPaymentRepository(db)
	return NewGormAdminStatsRepo(db, payments), db
}

// seedSpecialist insere um specialist via raw SQL e retorna o ID.
func seedSpecialist(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO specialists (id, name, description, prompt, status, created_at, updated_at)
		VALUES (?, ?, '', 'p', 'active', NOW(), NOW())`,
		id, name).Error
	require.NoError(t, err)
	return id
}

// linkSpecialistToTenant insere a linha de associação na junction table specialist_tenants.
func linkSpecialistToTenant(t *testing.T, db *gorm.DB, specialistID, tenantID string) {
	t.Helper()
	err := db.Exec(`INSERT INTO specialist_tenants (specialist_id, tenant_id, created_at)
		VALUES (?, ?, NOW())`,
		specialistID, tenantID).Error
	require.NoError(t, err)
}

// seedTenantWithPlano insere um tenant com plano explícito (status='active', created_at=NOW()).
// Necessário porque seedTenant sempre usa default plano='mensal'; aqui os testes do bloco
// financeiro precisam variar o plano para validar PlanDistribution.
func seedTenantWithPlano(t *testing.T, db *gorm.DB, name, plano string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO tenants
		(id, name, type, document, status, plano, exibir_pagamentos, created_at, updated_at)
		VALUES (?, ?, 'PJ', ?, 'active', ?, 1, NOW(), NOW())`,
		id, name, id[:20], plano).Error
	require.NoError(t, err)
	return id
}

// paymentOpts agrupa os campos necessários para INSERT direto em payments.
// Usamos SQL bruto (não domain.NewAvulsoPayment) para fixar status/datas em
// estados pré-computados que GlobalSummary precisa para o ano atual.
type paymentOpts struct {
	tenantID       string
	tipo           string // "avulso" ou "recorrente"
	descricao      string
	valorCents     int64
	status         string // "pendente" | "pago" | "atrasado" | "cancelado"
	dataVencimento time.Time
	dataPagamento  *time.Time
	createdAt      time.Time
}

func seedPayment(t *testing.T, db *gorm.DB, opts paymentOpts) string {
	t.Helper()
	id := uuid.New().String()
	var pagoArg interface{}
	if opts.dataPagamento != nil {
		pagoArg = *opts.dataPagamento
	}
	err := db.Exec(`INSERT INTO payments (id, tenant_id, tipo, descricao, valor_cents, status,
		data_vencimento, data_pagamento, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, opts.tenantID, opts.tipo, opts.descricao, opts.valorCents, opts.status,
		opts.dataVencimento, pagoArg, opts.createdAt, opts.createdAt).Error
	require.NoError(t, err)
	return id
}

// seedTenantWithStatus insere um tenant via raw SQL com status e created_at controlados.
// Necessário para os testes do AdminStatsRepo, que precisam fixar created_at no passado
// e variar o status — o repositório/dom\u00ednio padr\u00e3o n\u00e3o permite isso direto.
func seedTenantWithStatus(t *testing.T, db *gorm.DB, name, status string, createdAt time.Time) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO tenants
		(id, name, type, document, status, plano, exibir_pagamentos, created_at, updated_at)
		VALUES (?, ?, 'PJ', ?, ?, 'mensal', 1, ?, ?)`,
		id, name, id[:20], status, createdAt, createdAt).Error
	require.NoError(t, err)
	return id
}

// seedLeadsForTenant cria N leads para o tenant — cada um com seu próprio
// contact/conversation criados internamente por seedLead. Um \u00fanico funnel/coluna
// \u00e9 criado por chamada para reduzir overhead.
func seedLeadsForTenant(t *testing.T, db *gorm.DB, tenantID string, n int) {
	t.Helper()
	funnelID := seedFunnelDefault(t, db, tenantID, "F-"+uuid.New().String()[:6])
	colID := seedColumn(t, db, funnelID, "C", 0, "entry")
	for i := 0; i < n; i++ {
		seedLead(t, db, leadOpts{
			tenantID: tenantID, funnelID: funnelID, columnID: colID,
			status: "open", createdAt: time.Now(),
		})
	}
}

// ---------- TenantsBlock tests ----------

func TestTenantsBlock_ByStatusAndNewInMonth(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	// 3 active (1 criado este m\u00eas, 2 m\u00eas passado), 2 inactive, 1 blocked.
	seedTenantWithStatus(t, db, "T-active-old-1", "active", monthStart.AddDate(0, -1, 0))
	seedTenantWithStatus(t, db, "T-active-old-2", "active", monthStart.AddDate(0, -1, 0))
	seedTenantWithStatus(t, db, "T-active-new", "active", monthStart.AddDate(0, 0, 5))
	seedTenantWithStatus(t, db, "T-inactive-1", "inactive", monthStart.AddDate(0, -2, 0))
	seedTenantWithStatus(t, db, "T-inactive-2", "inactive", monthStart.AddDate(0, -2, 0))
	seedTenantWithStatus(t, db, "T-blocked", "blocked", monthStart.AddDate(0, -1, 0))

	got, err := repo.TenantsBlock(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, int64(3), got.Totals.Active)
	assert.Equal(t, int64(2), got.Totals.Inactive)
	assert.Equal(t, int64(1), got.Totals.Blocked)
	assert.Equal(t, int64(1), got.NewThisMonth)
}

func TestTenantsBlock_Last6Months(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// jan/26 (1), fev/26 (0), mar/26 (2), abr/26 (1) — backfill com 0 para os outros meses da janela.
	seedTenantWithStatus(t, db, "T-jan-1", "active", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	seedTenantWithStatus(t, db, "T-mar-1", "active", time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC))
	seedTenantWithStatus(t, db, "T-mar-2", "active", time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))
	seedTenantWithStatus(t, db, "T-abr-1", "active", time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	// fora da janela: out/2025 (a janela vai de nov/2025 at\u00e9 abr/2026).
	seedTenantWithStatus(t, db, "T-old", "active", time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC))

	got, err := repo.TenantsBlock(ctx, now)
	require.NoError(t, err)
	require.Len(t, got.Last6Months, 6)
	// Ordem cronol\u00f3gica: 2025-11, 2025-12, 2026-01, 2026-02, 2026-03, 2026-04
	assert.Equal(t, "2025-11", got.Last6Months[0].Label)
	assert.Equal(t, int64(0), got.Last6Months[0].Count)
	assert.Equal(t, "2025-12", got.Last6Months[1].Label)
	assert.Equal(t, int64(0), got.Last6Months[1].Count)
	assert.Equal(t, "2026-01", got.Last6Months[2].Label)
	assert.Equal(t, int64(1), got.Last6Months[2].Count)
	assert.Equal(t, "2026-02", got.Last6Months[3].Label)
	assert.Equal(t, int64(0), got.Last6Months[3].Count)
	assert.Equal(t, "2026-03", got.Last6Months[4].Label)
	assert.Equal(t, int64(2), got.Last6Months[4].Count)
	assert.Equal(t, "2026-04", got.Last6Months[5].Label)
	assert.Equal(t, int64(1), got.Last6Months[5].Count)
}

// ---------- UsageBlock tests ----------

func TestUsageBlock_Totals(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	contactID := seedContact(t, db, tenantID)
	funnelID := seedFunnelDefault(t, db, tenantID, "F")
	columnID := seedColumn(t, db, funnelID, "Novos", 0, "entry")

	// 3 leads (cada um cria contact+conversation pr\u00f3prios)
	for i := 0; i < 3; i++ {
		seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", createdAt: time.Now()})
	}

	// 2 conversations adicionais reaproveitando contactID acima:
	// 1 open (default) + 1 que vamos marcar como closed
	convOpen := seedConversation(t, db, tenantID, contactID)
	convClosed := seedConversation(t, db, tenantID, contactID)
	require.NoError(t, db.Exec(`UPDATE conversations SET status='closed' WHERE id = ?`, convClosed).Error)

	// 4 mensagens distribu\u00eddas
	now := time.Now()
	seedMessage(t, db, convOpen, "incoming", "x", now)
	seedMessage(t, db, convOpen, "outgoing", "y", now)
	seedMessage(t, db, convOpen, "incoming", "z", now)
	seedMessage(t, db, convClosed, "incoming", "w", now)

	got, err := repo.UsageBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), got.TotalLeads)
	assert.Equal(t, int64(4), got.TotalMessages)
	// 3 conversations open vindas dos seedLead + 1 convOpen aqui = 4 abertas;
	// convClosed est\u00e1 closed (n\u00e3o conta).
	assert.Equal(t, int64(4), got.ActiveConversations)
}

// ---------- HealthBlock tests ----------

func TestHealthBlock_Top10Active(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()

	t1 := seedTenant(t, db) // 3 leads
	t2 := seedTenant(t, db) // 1 lead
	t3 := seedTenant(t, db) // 5 leads
	seedLeadsForTenant(t, db, t1, 3)
	seedLeadsForTenant(t, db, t2, 1)
	seedLeadsForTenant(t, db, t3, 5)

	got, err := repo.HealthBlock(ctx, time.Now())
	require.NoError(t, err)
	require.Len(t, got.Top10Active, 3)
	// ordenado por COUNT(leads) DESC: t3(5) > t1(3) > t2(1)
	assert.Equal(t, t3, got.Top10Active[0].TenantID)
	assert.Equal(t, int64(5), got.Top10Active[0].LeadCount)
	assert.Equal(t, t1, got.Top10Active[1].TenantID)
	assert.Equal(t, int64(3), got.Top10Active[1].LeadCount)
	assert.Equal(t, t2, got.Top10Active[2].TenantID)
	assert.Equal(t, int64(1), got.Top10Active[2].LeadCount)
}

func TestHealthBlock_Top10Active_IncludesBlockedTenants(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()

	tActive := seedTenantWithStatus(t, db, "T-active", "active", time.Now().AddDate(0, 0, -90))
	tBlocked := seedTenantWithStatus(t, db, "T-blocked", "blocked", time.Now().AddDate(0, 0, -90))

	seedLeadsForTenant(t, db, tActive, 2)
	seedLeadsForTenant(t, db, tBlocked, 5) // mais leads que o ativo

	got, err := repo.HealthBlock(ctx, time.Now())
	require.NoError(t, err)
	require.Len(t, got.Top10Active, 2)
	// Blocked tenant deve aparecer em primeiro (5 leads > 2 leads), por design:
	// "active" = volume de leads, não status operacional.
	assert.Equal(t, tBlocked, got.Top10Active[0].TenantID)
	assert.Equal(t, int64(5), got.Top10Active[0].LeadCount)
	assert.Equal(t, tActive, got.Top10Active[1].TenantID)
}

func TestHealthBlock_InactiveOver30Days(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// Tenant ativo: criado h\u00e1 60d e tem lead recente (10d atr\u00e1s). N\u00c3O inativo.
	tActive := seedTenantWithStatus(t, db, "T-active", "active", now.AddDate(0, 0, -60))
	funnelA := seedFunnelDefault(t, db, tActive, "F")
	colA := seedColumn(t, db, funnelA, "C", 0, "entry")
	leadActive := seedLead(t, db, leadOpts{tenantID: tActive, funnelID: funnelA, columnID: colA, status: "open", createdAt: now.AddDate(0, 0, -10)})
	require.NoError(t, db.Exec(`UPDATE leads SET created_at=? WHERE id=?`, now.AddDate(0, 0, -10), leadActive).Error)

	// Tenant inativo: criado h\u00e1 60d, lead h\u00e1 50d. INATIVO (last_activity = lead.created_at = now-50d).
	tInactive := seedTenantWithStatus(t, db, "T-inactive", "active", now.AddDate(0, 0, -60))
	funnelI := seedFunnelDefault(t, db, tInactive, "F")
	colI := seedColumn(t, db, funnelI, "C", 0, "entry")
	leadInactive := seedLead(t, db, leadOpts{tenantID: tInactive, funnelID: funnelI, columnID: colI, status: "open", createdAt: now.AddDate(0, 0, -50)})
	require.NoError(t, db.Exec(`UPDATE leads SET created_at=? WHERE id=?`, now.AddDate(0, 0, -50), leadInactive).Error)

	// Tenant nunca teve lead: criado h\u00e1 60d. INATIVO via fallback em created_at.
	tEmpty := seedTenantWithStatus(t, db, "T-empty", "active", now.AddDate(0, 0, -60))

	// Tenant criado h\u00e1 10d, sem lead. N\u00c3O inativo (last_activity = created_at = -10d, < cutoff 30d).
	seedTenantWithStatus(t, db, "T-new", "active", now.AddDate(0, 0, -10))

	got, err := repo.HealthBlock(ctx, now)
	require.NoError(t, err)
	require.Len(t, got.InactiveList, 2, "esperados 2 inativos: T-inactive (lead -50d) e T-empty (created -60d)")

	// Ordem por last_activity ASC (mais antigo primeiro): T-empty (-60d) antes de T-inactive (-50d).
	inactiveByID := map[string]int64{}
	for _, it := range got.InactiveList {
		inactiveByID[it.TenantID] = it.DaysInactive
	}
	require.Contains(t, inactiveByID, tInactive)
	require.Contains(t, inactiveByID, tEmpty)
	assert.NotContains(t, inactiveByID, tActive)
	assert.InDelta(t, 50, inactiveByID[tInactive], 2, "DaysInactive aprox. 50 (last_activity = lead -50d)")
	assert.InDelta(t, 60, inactiveByID[tEmpty], 2, "DaysInactive aprox. 60 (last_activity = created_at -60d)")
	// Ordena\u00e7\u00e3o: T-empty primeiro (mais antigo).
	assert.Equal(t, tEmpty, got.InactiveList[0].TenantID, "T-empty deve vir primeiro (last_activity mais antigo)")
}

// ---------- EspecialistasBlock tests ----------

func TestEspecialistasBlock_TotalAndByTenant(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()

	t1 := seedTenantWithStatus(t, db, "Tenant Um", "active", time.Now())
	t2 := seedTenantWithStatus(t, db, "Tenant Dois", "active", time.Now())
	s1 := seedSpecialist(t, db, "S1")
	s2 := seedSpecialist(t, db, "S2")
	s3 := seedSpecialist(t, db, "S3")
	// s1 + s2 -> t1 (2 especialistas para t1)
	linkSpecialistToTenant(t, db, s1, t1)
	linkSpecialistToTenant(t, db, s2, t1)
	// s3 -> t2 (1 especialista para t2)
	linkSpecialistToTenant(t, db, s3, t2)

	got, err := repo.EspecialistasBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), got.Total)
	assert.Equal(t, int64(0), got.Qualifications) // por enquanto sempre 0
	require.Len(t, got.ByTenant, 2)
	// Ordenado desc por total: t1(2) antes de t2(1).
	assert.Equal(t, t1, got.ByTenant[0].TenantID)
	assert.Equal(t, "Tenant Um", got.ByTenant[0].TenantName)
	assert.Equal(t, int64(2), got.ByTenant[0].Total)
	assert.Equal(t, t2, got.ByTenant[1].TenantID)
	assert.Equal(t, int64(1), got.ByTenant[1].Total)
}

func TestEspecialistasBlock_QualificationsAlwaysZero(t *testing.T) {
	// Documenta o comportamento atual (TODO F05): Qualifications fica em 0
	// até o módulo de qualificações expor o contador. Captura regressão se
	// alguém adicionar mock/contagem prematura.
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	seedSpecialist(t, db, "S1") // só pra ter contagem != 0

	got, err := repo.EspecialistasBlock(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.Qualifications, "Qualifications fica em 0 ate F05 expor contador")
}

// ---------- FinanceiroBlock tests ----------

func TestFinanceiroBlock_AggregatesFromPaymentRepoAndPlanDistribution(t *testing.T) {
	repo, db := setupAdminRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)

	// Tenants com diferentes planos:
	// 2 mensal, 1 anual, 1 vitalicio, 0 externo
	t1 := seedTenantWithPlano(t, db, "Mensal-1", "mensal")
	t2 := seedTenantWithPlano(t, db, "Mensal-2", "mensal")
	seedTenantWithPlano(t, db, "Anual-1", "anual")
	seedTenantWithPlano(t, db, "Vital-1", "vitalicio")

	// Pagamentos (alimentam GlobalSummary):
	// t1: pago 200000 no ano + pendente 50000 + atrasado 30000
	// t2: atrasado 80000
	paid := now
	seedPayment(t, db, paymentOpts{tenantID: t1, tipo: "avulso", descricao: "x", valorCents: 200000, status: "pago", dataVencimento: now, dataPagamento: &paid, createdAt: now})
	seedPayment(t, db, paymentOpts{tenantID: t1, tipo: "avulso", descricao: "y", valorCents: 50000, status: "pendente", dataVencimento: now, createdAt: now})
	seedPayment(t, db, paymentOpts{tenantID: t1, tipo: "avulso", descricao: "z", valorCents: 30000, status: "atrasado", dataVencimento: now, createdAt: now})
	seedPayment(t, db, paymentOpts{tenantID: t2, tipo: "avulso", descricao: "a", valorCents: 80000, status: "atrasado", dataVencimento: now, createdAt: now})

	got, err := repo.FinanceiroBlock(ctx, now)
	require.NoError(t, err)

	// Vindos do GlobalSummary
	assert.Equal(t, int64(200000), got.ReceitaAnoCents)
	assert.Equal(t, int64(50000), got.PendenteTotalCents)
	assert.Equal(t, int64(110000), got.AtrasadoTotalCents) // 30000 + 80000

	// Distribuição de planos (vinda da query direta em tenants.plano)
	assert.Equal(t, int64(2), got.PlanDist.Mensal)
	assert.Equal(t, int64(1), got.PlanDist.Anual)
	assert.Equal(t, int64(1), got.PlanDist.Vitalicio)
	assert.Equal(t, int64(0), got.PlanDist.Externo)

	// TopOverdue (vindo do GlobalSummary, ordenado por valor desc)
	require.Len(t, got.TopOverdue, 2)
	assert.Equal(t, t2, got.TopOverdue[0].TenantID) // 80000 vem antes de 30000
	assert.Equal(t, int64(80000), got.TopOverdue[0].ValorCents)
	assert.Equal(t, "Mensal-2", got.TopOverdue[0].TenantName)
}

func TestFinanceiroBlock_EmptyPlatform(t *testing.T) {
	repo, _ := setupAdminRepo(t)
	ctx := context.Background()
	got, err := repo.FinanceiroBlock(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(0), got.ReceitaAnoCents)
	assert.Equal(t, int64(0), got.PendenteTotalCents)
	assert.Equal(t, int64(0), got.AtrasadoTotalCents)
	assert.Equal(t, int64(0), got.PlanDist.Mensal)
	assert.Empty(t, got.TopOverdue)
}
