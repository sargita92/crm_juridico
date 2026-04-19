package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAdminRepo reaproveita o cleanup/migrations de setupStatsRepo e devolve o
// repositório admin sobre o mesmo *gorm.DB já limpo.
func setupAdminRepo(t *testing.T) (*GormAdminStatsRepo, *gorm.DB) {
	t.Helper()
	_, db := setupStatsRepo(t)
	return NewGormAdminStatsRepo(db), db
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
