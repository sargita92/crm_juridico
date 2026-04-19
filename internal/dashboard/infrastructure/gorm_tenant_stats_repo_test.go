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
