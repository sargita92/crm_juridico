package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

func setupStatsRepo(t *testing.T) (*GormTenantStatsRepo, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	// Clean tables — order matters due to FK constraints.
	db.Exec("DELETE FROM lead_notes")
	db.Exec("DELETE FROM lead_movements")
	db.Exec("DELETE FROM leads")
	db.Exec("DELETE FROM funnel_columns")
	db.Exec("DELETE FROM funnel_products")
	db.Exec("DELETE FROM group_funnels")
	db.Exec("DELETE FROM funnels")
	db.Exec("DELETE FROM messages")
	db.Exec("DELETE FROM conversations")
	db.Exec("DELETE FROM contacts")
	db.Exec("DELETE FROM tenant_products")
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM payments")
	db.Exec("DELETE FROM tenants")

	return NewGormTenantStatsRepo(db), db
}

// seedTenant inserts a tenant via the existing repository.
func seedTenant(t *testing.T, db *gorm.DB) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tenant, err := tenantDomain.NewTenant(uuid.New().String(), "Escritorio Teste", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tenant))
	return tenant.ID
}

// seedFunnelDefault inserts a default funnel for the tenant via raw SQL and returns its ID.
func seedFunnelDefault(t *testing.T, db *gorm.DB, tenantID, name string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO funnels (id, tenant_id, name, description, active, is_default, created_at, updated_at)
		VALUES (?, ?, ?, '', TRUE, TRUE, NOW(), NOW())`,
		id, tenantID, name).Error
	require.NoError(t, err)
	return id
}

// seedColumn inserts a funnel_column via raw SQL and returns its ID.
func seedColumn(t *testing.T, db *gorm.DB, funnelID, name string, orderIdx int, colType string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO funnel_columns (id, funnel_id, name, order_index, type, color, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, '#3b82f6', NOW(), NOW())`,
		id, funnelID, name, orderIdx, colType).Error
	require.NoError(t, err)
	return id
}

// seedContact inserts a contact via raw SQL and returns its ID.
func seedContact(t *testing.T, db *gorm.DB, tenantID string) string {
	t.Helper()
	id := uuid.New().String()
	whatsappID := id + "@s.whatsapp.net"
	err := db.Exec(`INSERT INTO contacts (id, tenant_id, name, phone, whatsapp_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`,
		id, tenantID, "Contato "+id[:8], "5511"+id[:9], whatsappID).Error
	require.NoError(t, err)
	return id
}

// seedConversation inserts a conversation via raw SQL and returns its ID.
func seedConversation(t *testing.T, db *gorm.DB, tenantID, contactID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at)
		VALUES (?, ?, ?, 'open', NOW(), 0, NOW(), NOW())`,
		id, tenantID, contactID).Error
	require.NoError(t, err)
	return id
}

// seedProduct inserts a global product and the tenant_products association row, returning the product ID.
// Note: products is global (no tenant_id) since migration 000028; tenant association lives in tenant_products.
func seedProduct(t *testing.T, db *gorm.DB, tenantID, name string) string {
	t.Helper()
	productID := uuid.New().String()
	err := db.Exec(`INSERT INTO products (id, name, active, created_at, updated_at)
		VALUES (?, ?, 1, NOW(), NOW())`,
		productID, name).Error
	require.NoError(t, err)

	assocID := uuid.New().String()
	err = db.Exec(`INSERT INTO tenant_products (id, tenant_id, product_id, created_at) VALUES (?, ?, ?, NOW())`,
		assocID, tenantID, productID).Error
	require.NoError(t, err)
	return productID
}

type leadOpts struct {
	tenantID    string
	funnelID    string
	columnID    string
	status      string // "open" | "won" | "lost"
	productID   *string
	responsible *string
	createdAt   time.Time // if zero, NOW() is used
}

// seedLead inserts a lead and (if needed) its contact + conversation. Returns the lead ID.
func seedLead(t *testing.T, db *gorm.DB, opts leadOpts) string {
	t.Helper()
	contactID := seedContact(t, db, opts.tenantID)
	conversationID := seedConversation(t, db, opts.tenantID, contactID)
	id := uuid.New().String()

	createdAt := opts.createdAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	err := db.Exec(`INSERT INTO leads
		(id, tenant_id, funnel_id, column_id, contact_id, conversation_id, score, status, column_entered_at, created_at, updated_at, product_id, responsible_user_id)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)`,
		id, opts.tenantID, opts.funnelID, opts.columnID, contactID, conversationID,
		opts.status, createdAt, createdAt, createdAt,
		opts.productID, opts.responsible).Error
	require.NoError(t, err)
	return id
}

// ---------- FunilBlock tests ----------

func TestFunilBlock_OwnerScope(t *testing.T) {
	repo, db := setupStatsRepo(t)
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil Padrao")
	colEntry := seedColumn(t, db, funnelID, "Entrada", 0, "entry")
	colWon := seedColumn(t, db, funnelID, "Ganhou", 1, "won")

	// 3 open na col entry
	for i := 0; i < 3; i++ {
		seedLead(t, db, leadOpts{
			tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		})
	}
	// 1 won na col won
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colWon, status: "won",
	})
	// 1 lost na col entry
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "lost",
	})

	block, funnelName, err := repo.FunilBlock(context.Background(), tenantID, nil, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "Funil Padrao", funnelName)

	assert.Equal(t, int64(3), block.StatusTotals.Open)
	assert.Equal(t, int64(1), block.StatusTotals.Won)
	assert.Equal(t, int64(1), block.StatusTotals.Lost)
	assert.InDelta(t, 50.0, block.ConversionPct, 0.0001)

	// 2 columns with leads
	require.Len(t, block.ColumnTotals, 2)
	// Order is by order_index ASC
	assert.Equal(t, colEntry, block.ColumnTotals[0].ColumnID)
	assert.Equal(t, "Entrada", block.ColumnTotals[0].ColumnName)
	assert.Equal(t, 0, block.ColumnTotals[0].OrderIndex)
	assert.Equal(t, int64(4), block.ColumnTotals[0].Count) // 3 open + 1 lost

	assert.Equal(t, colWon, block.ColumnTotals[1].ColumnID)
	assert.Equal(t, "Ganhou", block.ColumnTotals[1].ColumnName)
	assert.Equal(t, 1, block.ColumnTotals[1].OrderIndex)
	assert.Equal(t, int64(1), block.ColumnTotals[1].Count)
}

func TestFunilBlock_UserScope(t *testing.T) {
	repo, db := setupStatsRepo(t)
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil X")
	colEntry := seedColumn(t, db, funnelID, "Entrada", 0, "entry")
	colWon := seedColumn(t, db, funnelID, "Ganhou", 1, "won")

	u1 := uuid.New().String()
	u2 := uuid.New().String()

	// u1: 2 open + 1 won
	for i := 0; i < 2; i++ {
		seedLead(t, db, leadOpts{
			tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open", responsible: &u1,
		})
	}
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colWon, status: "won", responsible: &u1,
	})

	// u2: 1 open + 1 lost
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open", responsible: &u2,
	})
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "lost", responsible: &u2,
	})

	block, _, err := repo.FunilBlock(context.Background(), tenantID, &u1, time.Now())
	require.NoError(t, err)

	assert.Equal(t, int64(2), block.StatusTotals.Open)
	assert.Equal(t, int64(1), block.StatusTotals.Won)
	assert.Equal(t, int64(0), block.StatusTotals.Lost)
	assert.InDelta(t, 100.0, block.ConversionPct, 0.0001)

	// Both columns with u1 leads (entry has 2 open; won has 1 won)
	require.Len(t, block.ColumnTotals, 2)
	totalsByCol := map[string]int64{}
	for _, c := range block.ColumnTotals {
		totalsByCol[c.ColumnID] = c.Count
	}
	assert.Equal(t, int64(2), totalsByCol[colEntry])
	assert.Equal(t, int64(1), totalsByCol[colWon])
}

func TestFunilBlock_NewTodayAndWeek(t *testing.T) {
	repo, db := setupStatsRepo(t)
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil Tempo")
	colEntry := seedColumn(t, db, funnelID, "Entrada", 0, "entry")

	// Use a fixed `now` that is a Wednesday so that now-2d is in the same week (Sunday-start week).
	// Wednesday 2025-04-16 12:00:00 UTC. Local() to play nice with the MySQL TIMESTAMP local conversion.
	// 12:00 escolhido para dar >=12h de cushion contra TZ skew Go-vs-MySQL.
	// Quarta-feira 16/abr/2025 mantém now-2d (segunda) na mesma semana Sun-start.
	now := time.Date(2025, 4, 16, 12, 0, 0, 0, time.Local)

	// 1 lead created today
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		createdAt: now,
	})
	// 1 lead created 2 days ago (Monday) — same week as Wednesday under Sunday-start
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		createdAt: now.AddDate(0, 0, -2),
	})
	// 1 lead created 10 days ago — outside the week
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		createdAt: now.AddDate(0, 0, -10),
	})

	block, _, err := repo.FunilBlock(context.Background(), tenantID, nil, now)
	require.NoError(t, err)

	assert.Equal(t, int64(1), block.NewToday)
	assert.Equal(t, int64(2), block.NewThisWeek)
}

// ---------- ProdutosBlock tests ----------

func TestProdutosBlock_OwnerScope(t *testing.T) {
	repo, db := setupStatsRepo(t)
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil P")
	colEntry := seedColumn(t, db, funnelID, "Entrada", 0, "entry")
	colWon := seedColumn(t, db, funnelID, "Ganhou", 1, "won")
	colLost := seedColumn(t, db, funnelID, "Perdeu", 2, "lost")

	p1 := seedProduct(t, db, tenantID, "Produto A")
	p2 := seedProduct(t, db, tenantID, "Produto B")

	// p1: 1 won + 1 open
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colWon, status: "won", productID: &p1,
	})
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open", productID: &p1,
	})
	// p2: 1 lost + 1 open
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colLost, status: "lost", productID: &p2,
	})
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open", productID: &p2,
	})

	rows, err := repo.ProdutosBlock(context.Background(), tenantID, nil)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := map[string]int{}
	for i, r := range rows {
		byID[r.ProductID] = i
	}
	require.Contains(t, byID, p1)
	require.Contains(t, byID, p2)

	row1 := rows[byID[p1]]
	assert.Equal(t, "Produto A", row1.ProductName)
	assert.Equal(t, int64(2), row1.Total)
	assert.Equal(t, int64(1), row1.Won)
	assert.Equal(t, int64(0), row1.Lost)

	row2 := rows[byID[p2]]
	assert.Equal(t, "Produto B", row2.ProductName)
	assert.Equal(t, int64(2), row2.Total)
	assert.Equal(t, int64(0), row2.Won)
	assert.Equal(t, int64(1), row2.Lost)
}

func TestProdutosBlock_UserScope(t *testing.T) {
	repo, db := setupStatsRepo(t)
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil PU")
	colEntry := seedColumn(t, db, funnelID, "Entrada", 0, "entry")

	p1 := seedProduct(t, db, tenantID, "Produto u1")
	p2 := seedProduct(t, db, tenantID, "Produto u2")

	u1 := uuid.New().String()
	u2 := uuid.New().String()

	// u1 leads only on p1
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		productID: &p1, responsible: &u1,
	})
	// u2 leads only on p2
	seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: colEntry, status: "open",
		productID: &p2, responsible: &u2,
	})

	rows, err := repo.ProdutosBlock(context.Background(), tenantID, &u1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, p1, rows[0].ProductID)
	assert.Equal(t, "Produto u1", rows[0].ProductName)
	assert.Equal(t, int64(1), rows[0].Total)
}

// ---------- WhatsAppBlock helpers ----------

// seedMessage inserts a message row and returns its ID. timestamp e direction são explícitos.
func seedMessage(t *testing.T, db *gorm.DB, conversationID, direction, content string, ts time.Time) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO messages (id, conversation_id, direction, content, type, status, timestamp, created_at)
		VALUES (?, ?, ?, ?, 'text', 'sent', ?, NOW())`,
		id, conversationID, direction, content, ts).Error
	require.NoError(t, err)
	return id
}

// leadConversationID returns the conversation_id linked to a previously seeded lead.
// seedLead cria a conversa internamente; este helper expõe esse ID para testes que
// precisam vincular mensagens ao mesmo lead/conversa.
func leadConversationID(t *testing.T, db *gorm.DB, leadID string) string {
	t.Helper()
	var convID string
	err := db.Raw(`SELECT conversation_id FROM leads WHERE id = ?`, leadID).Scan(&convID).Error
	require.NoError(t, err)
	require.NotEmpty(t, convID)
	return convID
}

// seedUser inserts a user row and returns its ID. Usado para o JOIN com users.name em ResponsaveisBlock.
func seedUser(t *testing.T, db *gorm.DB, name string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, 'x', 'user', 'active', NOW(), NOW())`,
		id, name, id+"@test.local").Error
	require.NoError(t, err)
	return id
}

// ---------- WhatsAppBlock tests ----------

func TestWhatsAppBlock_CountsAndActive(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	contactID := seedContact(t, db, tenantID)

	// 2 abertas, 3 fechadas — total 5 conversas
	open1 := seedConversation(t, db, tenantID, contactID)
	open2 := seedConversation(t, db, tenantID, contactID)
	closed1 := seedConversation(t, db, tenantID, contactID)
	closed2 := seedConversation(t, db, tenantID, contactID)
	closed3 := seedConversation(t, db, tenantID, contactID)
	// Marcar 3 como closed
	require.NoError(t, db.Exec(`UPDATE conversations SET status='closed' WHERE id IN (?, ?, ?)`, closed1, closed2, closed3).Error)

	// 10 incoming espalhadas; 7 outgoing
	base := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		seedMessage(t, db, open1, "incoming", "in", base.Add(time.Duration(i)*time.Minute))
	}
	for i := 0; i < 5; i++ {
		seedMessage(t, db, open2, "incoming", "in", base.Add(time.Duration(i)*time.Minute))
	}
	for i := 0; i < 4; i++ {
		seedMessage(t, db, open1, "outgoing", "out", base.Add(time.Duration(10+i)*time.Minute))
	}
	for i := 0; i < 3; i++ {
		seedMessage(t, db, open2, "outgoing", "out", base.Add(time.Duration(10+i)*time.Minute))
	}

	got, err := repo.WhatsAppBlock(ctx, tenantID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), got.ActiveConversations)
	assert.Equal(t, int64(10), got.IncomingMessages)
	assert.Equal(t, int64(7), got.OutgoingMessages)
}

func TestWhatsAppBlock_FirstResponseAvg(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	contactID := seedContact(t, db, tenantID)

	// Conversa A: incoming 10:00, outgoing 10:05 → 300s
	convA := seedConversation(t, db, tenantID, contactID)
	seedMessage(t, db, convA, "incoming", "x", time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC))
	seedMessage(t, db, convA, "outgoing", "y", time.Date(2026, 4, 19, 10, 5, 0, 0, time.UTC))

	// Conversa B: incoming 11:00, outgoing 11:02 → 120s
	convB := seedConversation(t, db, tenantID, contactID)
	seedMessage(t, db, convB, "incoming", "x", time.Date(2026, 4, 19, 11, 0, 0, 0, time.UTC))
	seedMessage(t, db, convB, "outgoing", "y", time.Date(2026, 4, 19, 11, 2, 0, 0, time.UTC))

	got, err := repo.WhatsAppBlock(ctx, tenantID, nil)
	require.NoError(t, err)
	assert.InDelta(t, 210.0, got.FirstResponseAvgSec, 0.5) // (300+120)/2 = 210
}

func TestWhatsAppBlock_UserScope_FiltersByLeadResponsible(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "Funil")
	columnID := seedColumn(t, db, funnelID, "Novos", 0, "entry")

	// Lead A → responsável u2 (seedLead cria contact + conversation internamente)
	u2 := uuid.New().String()
	leadA := seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: columnID,
		status: "open", responsible: &u2,
	})
	convA := leadConversationID(t, db, leadA)

	// Lead B → responsável "other" (contact distinto via seedLead, sem violar UNIQUE(tenant_id, contact_id))
	other := uuid.New().String()
	leadB := seedLead(t, db, leadOpts{
		tenantID: tenantID, funnelID: funnelID, columnID: columnID,
		status: "open", responsible: &other,
	})
	convB := leadConversationID(t, db, leadB)

	seedMessage(t, db, convA, "incoming", "x", time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC))
	seedMessage(t, db, convA, "outgoing", "y", time.Date(2026, 4, 19, 9, 1, 0, 0, time.UTC))
	seedMessage(t, db, convB, "incoming", "x", time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC))
	seedMessage(t, db, convB, "outgoing", "y", time.Date(2026, 4, 19, 9, 1, 0, 0, time.UTC))

	got, err := repo.WhatsAppBlock(ctx, tenantID, &u2)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.ActiveConversations) // só convA pertence a u2
	assert.Equal(t, int64(1), got.IncomingMessages)
	assert.Equal(t, int64(1), got.OutgoingMessages)
	assert.InDelta(t, 60.0, got.FirstResponseAvgSec, 0.5)
}

// ---------- ResponsaveisBlock tests ----------

func TestResponsaveisBlock_Owner(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "F")
	columnID := seedColumn(t, db, funnelID, "Novos", 0, "entry")

	u1 := seedUser(t, db, "Maria")
	u2 := seedUser(t, db, "João")
	u3 := seedUser(t, db, "Ana")

	// 5 leads para u1 (3 open, 1 won, 1 lost)
	for i := 0; i < 3; i++ {
		seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: &u1})
	}
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "won", responsible: &u1})
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "lost", responsible: &u1})
	// 3 leads para u2 (1 open, 2 won)
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: &u2})
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "won", responsible: &u2})
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "won", responsible: &u2})
	// 1 lead para u3
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: &u3})
	// 1 lead sem responsible (não deve aparecer)
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: nil})

	got, err := repo.ResponsaveisBlock(ctx, tenantID, nil)
	require.NoError(t, err)
	require.Len(t, got, 3)
	// ordenado por total DESC: u1(5), u2(3), u3(1)
	assert.Equal(t, u1, got[0].UserID)
	assert.Equal(t, "Maria", got[0].UserName)
	assert.Equal(t, int64(5), got[0].Total)
	assert.Equal(t, int64(1), got[0].Won)
	assert.Equal(t, int64(1), got[0].Lost)
	assert.Equal(t, u2, got[1].UserID)
	assert.Equal(t, int64(3), got[1].Total)
	assert.Equal(t, int64(2), got[1].Won)
	assert.Equal(t, u3, got[2].UserID)
	assert.Equal(t, int64(1), got[2].Total)
}

func TestResponsaveisBlock_UserScope(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "F")
	columnID := seedColumn(t, db, funnelID, "Novos", 0, "entry")
	u1 := seedUser(t, db, "Maria")
	u2 := seedUser(t, db, "João")

	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: &u1})
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "won", responsible: &u1})
	seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: columnID, status: "open", responsible: &u2})

	got, err := repo.ResponsaveisBlock(ctx, tenantID, &u1)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, u1, got[0].UserID)
	assert.Equal(t, int64(2), got[0].Total)
	assert.Equal(t, int64(1), got[0].Won)
}

// ---------- TempoFunilBlock tests ----------

func TestTempoFunilBlock_AvgHours(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "F")
	colA := seedColumn(t, db, funnelID, "A", 0, "entry")
	colB := seedColumn(t, db, funnelID, "B", 1, "intermediate")

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	// 2 leads na coluna A: column_entered_at = now-2h e now-4h → media 3h
	leadA1 := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colA, status: "open", createdAt: now})
	leadA2 := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colA, status: "open", createdAt: now})
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-2*time.Hour), leadA1).Error)
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-4*time.Hour), leadA2).Error)
	// 1 lead na coluna B com 1h
	leadB := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colB, status: "open", createdAt: now})
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-1*time.Hour), leadB).Error)

	got, err := repo.TempoFunilBlock(ctx, tenantID, nil, now)
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Ordenado por order_index: A(0) primeiro, B(1) depois
	assert.Equal(t, colA, got[0].ColumnID)
	assert.InDelta(t, 3.0, got[0].AvgHours, 0.01)
	assert.Equal(t, int64(0), got[0].StuckOver7Days)
	assert.Equal(t, colB, got[1].ColumnID)
	assert.InDelta(t, 1.0, got[1].AvgHours, 0.01)
}

func TestTempoFunilBlock_StuckOver7Days(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	funnelID := seedFunnelDefault(t, db, tenantID, "F")
	colA := seedColumn(t, db, funnelID, "A", 0, "entry")

	now := time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)
	// Lead aberto há 10 dias
	leadStuck := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colA, status: "open", createdAt: now})
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-10*24*time.Hour), leadStuck).Error)
	// Lead aberto há 3 dias (não stuck)
	leadFresh := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colA, status: "open", createdAt: now})
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-3*24*time.Hour), leadFresh).Error)
	// Lead pago há 10 dias (won → não stuck mesmo se velho)
	leadWon := seedLead(t, db, leadOpts{tenantID: tenantID, funnelID: funnelID, columnID: colA, status: "won", createdAt: now})
	require.NoError(t, db.Exec(`UPDATE leads SET column_entered_at = ? WHERE id = ?`, now.Add(-10*24*time.Hour), leadWon).Error)

	got, err := repo.TempoFunilBlock(ctx, tenantID, nil, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, int64(1), got[0].StuckOver7Days) // só leadStuck
}

func TestResponsaveisBlock_EmptyTenant(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)

	got, err := repo.ResponsaveisBlock(ctx, tenantID, nil)
	require.NoError(t, err)
	assert.NotNil(t, got, "deve retornar slice vazio (não nil) para handler iterar")
	assert.Empty(t, got)
}

func TestTempoFunilBlock_EmptyTenant(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	// Tenant sem default funnel → método curto-circuita e retorna (nil, nil).
	// Esse caso é distinto de "tem funnel mas sem leads" (que retornaria slice vazio).

	got, err := repo.TempoFunilBlock(ctx, tenantID, nil, time.Now())
	require.NoError(t, err)
	assert.Nil(t, got, "sem default funnel → retorna nil (não slice vazio); handler deve tratar como empty state")
}

func TestTempoFunilBlock_FunnelExists_NoLeads(t *testing.T) {
	repo, db := setupStatsRepo(t)
	ctx := context.Background()
	tenantID := seedTenant(t, db)
	seedFunnelDefault(t, db, tenantID, "Funil")
	// Funnel existe mas sem leads → contraste com TestTempoFunilBlock_EmptyTenant.

	got, err := repo.TempoFunilBlock(ctx, tenantID, nil, time.Now())
	require.NoError(t, err)
	assert.NotNil(t, got, "deve retornar slice vazio (não nil) quando funnel existe sem leads")
	assert.Empty(t, got)
}
