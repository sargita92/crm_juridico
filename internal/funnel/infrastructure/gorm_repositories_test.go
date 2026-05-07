package infrastructure

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
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

type funnelTestRepos struct {
	funnels   *GormFunnelRepository
	columns   *GormColumnRepository
	leads     *GormLeadRepository
	movements *GormLeadMovementRepository
	notes     *GormLeadNoteRepository
}

func setupFunnelRepos(t *testing.T) (*funnelTestRepos, *gorm.DB) {
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
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return &funnelTestRepos{
		funnels:   NewGormFunnelRepository(db),
		columns:   NewGormColumnRepository(db),
		leads:     NewGormLeadRepository(db),
		movements: NewGormLeadMovementRepository(db),
		notes:     NewGormLeadNoteRepository(db),
	}, db
}

// createTenant inserts a tenant row and returns its ID.
func createTenant(t *testing.T, db *gorm.DB) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tenant, err := tenantDomain.NewTenant(uuid.New().String(), "Escritorio Teste", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tenant))
	return tenant.ID
}

// createContact inserts a contact row via raw SQL (no whatsapp dep) and returns its ID.
func createContact(t *testing.T, db *gorm.DB, tenantID, name, phone string) string {
	t.Helper()
	id := uuid.New().String()
	whatsappID := id + "@s.whatsapp.net"
	err := db.Exec(`INSERT INTO contacts (id, tenant_id, name, phone, whatsapp_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(), NOW())`,
		id, tenantID, name, phone, whatsappID).Error
	require.NoError(t, err)
	return id
}

// createConversation inserts a conversation row via raw SQL and returns its ID.
func createConversation(t *testing.T, db *gorm.DB, tenantID, contactID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at)
		VALUES (?, ?, ?, 'open', NOW(), 0, NOW(), NOW())`,
		id, tenantID, contactID).Error
	require.NoError(t, err)
	return id
}

// mustCreateFunnel creates and persists a funnel.
func mustCreateFunnel(t *testing.T, repo *GormFunnelRepository, tenantID, name string) *domain.Funnel {
	t.Helper()
	f, err := domain.NewFunnel(uuid.New().String(), tenantID, name, "")
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), f))
	return f
}

// mustCreateColumn creates and persists a funnel column.
func mustCreateColumn(t *testing.T, repo *GormColumnRepository, funnelID, name string, orderIdx int, colType domain.ColumnType) *domain.Column {
	t.Helper()
	c, err := domain.NewColumn(uuid.New().String(), funnelID, name, orderIdx, colType, "#3b82f6")
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), c))
	return c
}

// ---------- FunnelRepository ----------

func TestGormFunnelRepository_Create_And_FindByID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	ctx := context.Background()

	f, err := domain.NewFunnel(uuid.New().String(), tenantID, "Funil Padrao", "desc")
	require.NoError(t, err)
	require.NoError(t, repos.funnels.Create(ctx, f))

	found, err := repos.funnels.FindByID(ctx, f.ID)
	require.NoError(t, err)
	assert.Equal(t, f.ID, found.ID)
	assert.Equal(t, tenantID, found.TenantID)
	assert.Equal(t, "Funil Padrao", found.Name)
	assert.Equal(t, "desc", found.Description)
	assert.True(t, found.Active)
	assert.False(t, found.IsDefault)
}

func TestGormFunnelRepository_FindByID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	_, err := repos.funnels.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestGormFunnelRepository_Update(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	ctx := context.Background()

	f := mustCreateFunnel(t, repos.funnels, tenantID, "Original")
	require.NoError(t, f.Update("Atualizado", "nova desc"))
	require.NoError(t, repos.funnels.Update(ctx, f))

	found, err := repos.funnels.FindByID(ctx, f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Atualizado", found.Name)
	assert.Equal(t, "nova desc", found.Description)
}

func TestGormFunnelRepository_FindByTenantID_ListAndTenantIsolation(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantA := createTenant(t, db)
	tenantB := createTenant(t, db)
	ctx := context.Background()

	mustCreateFunnel(t, repos.funnels, tenantA, "A1")
	mustCreateFunnel(t, repos.funnels, tenantA, "A2")
	mustCreateFunnel(t, repos.funnels, tenantB, "B1")

	listA, err := repos.funnels.FindByTenantID(ctx, tenantA)
	require.NoError(t, err)
	assert.Len(t, listA, 2)
	for _, f := range listA {
		assert.Equal(t, tenantA, f.TenantID)
	}

	listB, err := repos.funnels.FindByTenantID(ctx, tenantB)
	require.NoError(t, err)
	assert.Len(t, listB, 1)
	assert.Equal(t, "B1", listB[0].Name)
}

func TestGormFunnelRepository_FindByTenantID_Empty(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	list, err := repos.funnels.FindByTenantID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestGormFunnelRepository_FindDefaultByTenantID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	ctx := context.Background()

	// Non-default funnel first.
	mustCreateFunnel(t, repos.funnels, tenantID, "Secundario")

	// Default funnel.
	defaultFunnel := mustCreateFunnel(t, repos.funnels, tenantID, "Padrao")
	defaultFunnel.SetDefault()
	require.NoError(t, repos.funnels.Update(ctx, defaultFunnel))

	found, err := repos.funnels.FindDefaultByTenantID(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, defaultFunnel.ID, found.ID)
	assert.True(t, found.IsDefault)
}

func TestGormFunnelRepository_FindDefaultByTenantID_NotFound(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)

	mustCreateFunnel(t, repos.funnels, tenantID, "NaoDefault")

	_, err := repos.funnels.FindDefaultByTenantID(context.Background(), tenantID)
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

// ---------- ColumnRepository ----------

func TestGormColumnRepository_Create_And_FindByID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	c, err := domain.NewColumn(uuid.New().String(), funnel.ID, "Novos", 0, domain.ColumnTypeEntry, "#ff0000")
	require.NoError(t, err)
	require.NoError(t, repos.columns.Create(ctx, c))

	found, err := repos.columns.FindByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, c.ID, found.ID)
	assert.Equal(t, funnel.ID, found.FunnelID)
	assert.Equal(t, "Novos", found.Name)
	assert.Equal(t, 0, found.OrderIndex)
	assert.Equal(t, domain.ColumnTypeEntry, found.Type)
	assert.Equal(t, "#ff0000", found.Color)
}

func TestGormColumnRepository_FindByID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	_, err := repos.columns.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestGormColumnRepository_Update(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	c := mustCreateColumn(t, repos.columns, funnel.ID, "Original", 1, domain.ColumnTypeIntermediate)
	require.NoError(t, c.Update("Modificada", domain.ColumnTypeWon, "#00ff00"))
	require.NoError(t, repos.columns.Update(ctx, c))

	found, err := repos.columns.FindByID(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Modificada", found.Name)
	assert.Equal(t, domain.ColumnTypeWon, found.Type)
	assert.Equal(t, "#00ff00", found.Color)
}

func TestGormColumnRepository_Delete(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	c := mustCreateColumn(t, repos.columns, funnel.ID, "Deletable", 0, domain.ColumnTypeIntermediate)

	require.NoError(t, repos.columns.Delete(ctx, c.ID))

	_, err := repos.columns.FindByID(ctx, c.ID)
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestGormColumnRepository_Delete_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	err := repos.columns.Delete(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestGormColumnRepository_FindByFunnelID_Ordered(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	// Inserted out of order intentionally.
	mustCreateColumn(t, repos.columns, funnel.ID, "Terceira", 2, domain.ColumnTypeWon)
	mustCreateColumn(t, repos.columns, funnel.ID, "Primeira", 0, domain.ColumnTypeEntry)
	mustCreateColumn(t, repos.columns, funnel.ID, "Segunda", 1, domain.ColumnTypeIntermediate)

	cols, err := repos.columns.FindByFunnelID(ctx, funnel.ID)
	require.NoError(t, err)
	require.Len(t, cols, 3)
	assert.Equal(t, "Primeira", cols[0].Name)
	assert.Equal(t, "Segunda", cols[1].Name)
	assert.Equal(t, "Terceira", cols[2].Name)
}

func TestGormColumnRepository_FindEntryByFunnelID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	mustCreateColumn(t, repos.columns, funnel.ID, "Qualificando", 1, domain.ColumnTypeIntermediate)
	entry := mustCreateColumn(t, repos.columns, funnel.ID, "Entrada", 0, domain.ColumnTypeEntry)

	found, err := repos.columns.FindEntryByFunnelID(ctx, funnel.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, found.ID)
	assert.Equal(t, domain.ColumnTypeEntry, found.Type)
}

func TestGormColumnRepository_FindEntryByFunnelID_NotFound(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")

	mustCreateColumn(t, repos.columns, funnel.ID, "Qualif", 0, domain.ColumnTypeIntermediate)

	_, err := repos.columns.FindEntryByFunnelID(context.Background(), funnel.ID)
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestGormColumnRepository_CountByFunnelID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	count, err := repos.columns.CountByFunnelID(ctx, funnel.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	mustCreateColumn(t, repos.columns, funnel.ID, "A", 0, domain.ColumnTypeEntry)
	mustCreateColumn(t, repos.columns, funnel.ID, "B", 1, domain.ColumnTypeIntermediate)

	count, err = repos.columns.CountByFunnelID(ctx, funnel.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGormColumnRepository_GetMaxOrderIndex_Empty(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")

	maxIdx, err := repos.columns.GetMaxOrderIndex(context.Background(), funnel.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, maxIdx)
}

func TestGormColumnRepository_GetMaxOrderIndex_WithColumns(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")

	mustCreateColumn(t, repos.columns, funnel.ID, "A", 0, domain.ColumnTypeEntry)
	mustCreateColumn(t, repos.columns, funnel.ID, "B", 3, domain.ColumnTypeIntermediate)
	mustCreateColumn(t, repos.columns, funnel.ID, "C", 1, domain.ColumnTypeWon)

	maxIdx, err := repos.columns.GetMaxOrderIndex(context.Background(), funnel.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, maxIdx)
}

func TestGormColumnRepository_SwapOrder(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "F")
	ctx := context.Background()

	c1 := mustCreateColumn(t, repos.columns, funnel.ID, "A", 0, domain.ColumnTypeEntry)
	c2 := mustCreateColumn(t, repos.columns, funnel.ID, "B", 1, domain.ColumnTypeIntermediate)

	require.NoError(t, repos.columns.SwapOrder(ctx, c1.ID, 1, c2.ID, 0))

	found1, err := repos.columns.FindByID(ctx, c1.ID)
	require.NoError(t, err)
	found2, err := repos.columns.FindByID(ctx, c2.ID)
	require.NoError(t, err)

	assert.Equal(t, 1, found1.OrderIndex)
	assert.Equal(t, 0, found2.OrderIndex)
}

// ---------- LeadRepository ----------

// leadFixture sets up a tenant + funnel + entry column + contact + conversation
// and returns everything needed to create leads.
type leadFixture struct {
	tenantID       string
	funnel         *domain.Funnel
	entryColumn    *domain.Column
	contactID      string
	conversationID string
}

func newLeadFixture(t *testing.T, repos *funnelTestRepos, db *gorm.DB) *leadFixture {
	t.Helper()
	tenantID := createTenant(t, db)
	funnel := mustCreateFunnel(t, repos.funnels, tenantID, "Funil")
	entry := mustCreateColumn(t, repos.columns, funnel.ID, "Entrada", 0, domain.ColumnTypeEntry)
	contactID := createContact(t, db, tenantID, "Cliente", "+5511999990001")
	convID := createConversation(t, db, tenantID, contactID)
	return &leadFixture{
		tenantID:       tenantID,
		funnel:         funnel,
		entryColumn:    entry,
		contactID:      contactID,
		conversationID: convID,
	}
}

func TestGormLeadRepository_Create_And_FindByID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	found, err := repos.leads.FindByID(ctx, lead.ID)
	require.NoError(t, err)
	assert.Equal(t, lead.ID, found.ID)
	assert.Equal(t, fx.tenantID, found.TenantID)
	assert.Equal(t, fx.funnel.ID, found.FunnelID)
	assert.Equal(t, fx.entryColumn.ID, found.ColumnID)
	assert.Equal(t, fx.contactID, found.ContactID)
	assert.Equal(t, fx.conversationID, found.ConversationID)
	assert.Equal(t, domain.LeadStatusOpen, found.Status)
	assert.Equal(t, "", found.ProductID)
}

func TestGormLeadRepository_FindByID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	_, err := repos.leads.FindByID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGormLeadRepository_Update(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	won := mustCreateColumn(t, repos.columns, fx.funnel.ID, "Ganhos", 1, domain.ColumnTypeWon)

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	lead.MoveTo(won.ID, domain.ColumnTypeWon)
	lead.UpdateScore(85)
	require.NoError(t, repos.leads.Update(ctx, lead))

	found, err := repos.leads.FindByID(ctx, lead.ID)
	require.NoError(t, err)
	assert.Equal(t, won.ID, found.ColumnID)
	assert.Equal(t, domain.LeadStatusWon, found.Status)
	assert.Equal(t, 85, found.Score)
}

func TestGormLeadRepository_FindByConversationID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	found, err := repos.leads.FindByConversationID(ctx, fx.conversationID)
	require.NoError(t, err)
	assert.Equal(t, lead.ID, found.ID)
}

func TestGormLeadRepository_FindByConversationID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	_, err := repos.leads.FindByConversationID(context.Background(), uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGormLeadRepository_FindByContactAndTenant(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	found, err := repos.leads.FindByContactAndTenant(ctx, fx.tenantID, fx.contactID)
	require.NoError(t, err)
	assert.Equal(t, lead.ID, found.ID)
}

func TestGormLeadRepository_FindByContactAndTenant_NotFound(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)

	_, err := repos.leads.FindByContactAndTenant(context.Background(), fx.tenantID, uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGormLeadRepository_FindByFunnelID_ListAll(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	// Create two leads in the same funnel.
	contact2 := createContact(t, db, fx.tenantID, "Cliente 2", "+5511999990002")
	conv2 := createConversation(t, db, fx.tenantID, contact2)

	lead1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead1))
	// Ensure created_at ordering is deterministic.
	time.Sleep(10 * time.Millisecond)
	lead2, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, contact2, conv2)
	require.NoError(t, repos.leads.Create(ctx, lead2))

	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), list.Total)
	assert.Len(t, list.Leads, 2)
	assert.Equal(t, 1, list.Page)
	assert.Equal(t, 20, list.Limit)

	// Embedded contact/column data is populated.
	for _, lwc := range list.Leads {
		assert.NotEmpty(t, lwc.ContactName)
		assert.NotEmpty(t, lwc.ContactPhone)
		assert.Equal(t, "Entrada", lwc.ColumnName)
	}
}

func TestGormLeadRepository_FindByFunnelID_DefaultPageLimit(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	// Page=0, Limit=0 should apply defaults.
	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{})
	require.NoError(t, err)
	assert.Equal(t, 1, list.Page)
	assert.Equal(t, 20, list.Limit)
	assert.Equal(t, int64(1), list.Total)
}

func TestGormLeadRepository_FindByFunnelID_FilterBySearch(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db) // first contact "Cliente"
	ctx := context.Background()

	contact2 := createContact(t, db, fx.tenantID, "Joao Silva", "+5511888880002")
	conv2 := createConversation(t, db, fx.tenantID, contact2)

	lead1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead1))
	lead2, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, contact2, conv2)
	require.NoError(t, repos.leads.Create(ctx, lead2))

	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{Search: "Joao", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Leads, 1)
	assert.Equal(t, "Joao Silva", list.Leads[0].ContactName)
}

func TestGormLeadRepository_FindByFunnelID_FilterByColumn(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	secondCol := mustCreateColumn(t, repos.columns, fx.funnel.ID, "Qualificado", 1, domain.ColumnTypeIntermediate)
	contact2 := createContact(t, db, fx.tenantID, "Maria", "+5511999990099")
	conv2 := createConversation(t, db, fx.tenantID, contact2)

	l1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, l1))
	l2, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, secondCol.ID, contact2, conv2)
	require.NoError(t, repos.leads.Create(ctx, l2))

	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{ColumnID: secondCol.ID, Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Leads, 1)
	assert.Equal(t, l2.ID, list.Leads[0].Lead.ID)
}

func TestGormLeadRepository_FindByFunnelID_FilterByProduct(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	// Seed a product row matching the leads.product_id FK.
	// products no longer has tenant_id (migration 000028 moved it to tenant_products).
	productID := uuid.New().String()
	require.NoError(t, db.Exec(`INSERT INTO products (id, name, description, active, created_at, updated_at)
		VALUES (?, ?, ?, 1, NOW(), NOW())`,
		productID, "Prod A", "").Error)

	contact2 := createContact(t, db, fx.tenantID, "Outro", "+5511999990123")
	conv2 := createConversation(t, db, fx.tenantID, contact2)

	l1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	l1.SetProduct(productID)
	require.NoError(t, repos.leads.Create(ctx, l1))

	l2, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, contact2, conv2)
	require.NoError(t, repos.leads.Create(ctx, l2))

	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{ProductID: productID, Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Leads, 1)
	assert.Equal(t, l1.ID, list.Leads[0].Lead.ID)
}

func TestGormLeadRepository_FindByFunnelID_Pagination(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	// Create 5 leads.
	l1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, l1))
	for i := 0; i < 4; i++ {
		c := createContact(t, db, fx.tenantID, "N", "+55119999000"+string(rune('0'+i)))
		cv := createConversation(t, db, fx.tenantID, c)
		l, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, c, cv)
		require.NoError(t, repos.leads.Create(ctx, l))
	}

	list, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), list.Total)
	assert.Len(t, list.Leads, 2)

	list2, err := repos.leads.FindByFunnelID(ctx, fx.funnel.ID, domain.LeadFilter{Page: 3, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), list2.Total)
	assert.Len(t, list2.Leads, 1)
}

func TestGormLeadRepository_FindByFunnelID_TenantIsolation(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fxA := newLeadFixture(t, repos, db)
	fxB := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lA, _ := domain.NewLead(uuid.New().String(), fxA.tenantID, fxA.funnel.ID, fxA.entryColumn.ID, fxA.contactID, fxA.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lA))
	lB, _ := domain.NewLead(uuid.New().String(), fxB.tenantID, fxB.funnel.ID, fxB.entryColumn.ID, fxB.contactID, fxB.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lB))

	// Querying funnel A should return only tenant A's lead.
	list, err := repos.leads.FindByFunnelID(ctx, fxA.funnel.ID, domain.LeadFilter{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.Leads, 1)
	assert.Equal(t, lA.ID, list.Leads[0].Lead.ID)
}

func TestGormLeadRepository_CountByColumnID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	count, err := repos.leads.CountByColumnID(ctx, fx.entryColumn.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	l, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, l))

	count, err = repos.leads.CountByColumnID(ctx, fx.entryColumn.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestGormLeadRepository_FindByTenantAndSearch_WithQuery(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db) // contact "Cliente"
	ctx := context.Background()

	contact2 := createContact(t, db, fx.tenantID, "Joana Neves", "+5511777770002")
	conv2 := createConversation(t, db, fx.tenantID, contact2)

	l1, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, l1))
	l2, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, contact2, conv2)
	require.NoError(t, repos.leads.Create(ctx, l2))

	// Search matching contact name.
	out, err := repos.leads.FindByTenantAndSearch(ctx, fx.tenantID, "Joana", 20)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, l2.ID, out[0].ID)

	// Search matching phone.
	out, err = repos.leads.FindByTenantAndSearch(ctx, fx.tenantID, "77777", 20)
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, l2.ID, out[0].ID)
}

func TestGormLeadRepository_FindByTenantAndSearch_EmptyQueryReturnsAll(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	l, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, l))

	// Empty query: all tenant leads, default limit when <= 0.
	out, err := repos.leads.FindByTenantAndSearch(ctx, fx.tenantID, "", 0)
	require.NoError(t, err)
	assert.Len(t, out, 1)
}

func TestGormLeadRepository_FindByTenantAndSearch_TenantIsolation(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fxA := newLeadFixture(t, repos, db)
	fxB := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lA, _ := domain.NewLead(uuid.New().String(), fxA.tenantID, fxA.funnel.ID, fxA.entryColumn.ID, fxA.contactID, fxA.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lA))
	lB, _ := domain.NewLead(uuid.New().String(), fxB.tenantID, fxB.funnel.ID, fxB.entryColumn.ID, fxB.contactID, fxB.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lB))

	outA, err := repos.leads.FindByTenantAndSearch(ctx, fxA.tenantID, "", 20)
	require.NoError(t, err)
	require.Len(t, outA, 1)
	assert.Equal(t, lA.ID, outA[0].ID)
}

// ---------- LeadMovementRepository ----------

func TestGormLeadMovementRepository_Create_And_FindByLeadID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	won := mustCreateColumn(t, repos.columns, fx.funnel.ID, "Ganho", 1, domain.ColumnTypeWon)

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	// Initial movement with empty FromColumnID — must persist as NULL.
	m1 := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", fx.entryColumn.ID)
	require.NoError(t, repos.movements.Create(ctx, m1))

	// Delay > 1s so ORDER BY moved_at DESC is deterministic
	// (moved_at is TIMESTAMP with second-level precision).
	time.Sleep(1100 * time.Millisecond)

	m2 := domain.NewLeadMovement(uuid.New().String(), lead.ID, fx.entryColumn.ID, won.ID)
	require.NoError(t, repos.movements.Create(ctx, m2))

	list, err := repos.movements.FindByLeadID(ctx, lead.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// Sorted DESC by moved_at: m2 first.
	assert.Equal(t, m2.ID, list[0].ID)
	assert.Equal(t, fx.entryColumn.ID, list[0].FromColumnID)
	assert.Equal(t, won.ID, list[0].ToColumnID)
	// m1 had nil FromColumnID → empty string in domain.
	assert.Equal(t, m1.ID, list[1].ID)
	assert.Equal(t, "", list[1].FromColumnID)
}

func TestGormLeadMovementRepository_FindByLeadID_Empty(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	list, err := repos.movements.FindByLeadID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, list)
}

// ---------- LeadNoteRepository ----------

func TestGormLeadNoteRepository_Create_And_FindByLeadID(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	fx := newLeadFixture(t, repos, db)
	ctx := context.Background()

	lead, _ := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, repos.leads.Create(ctx, lead))

	creator := uuid.New().String()
	n1, err := domain.NewLeadNote(uuid.New().String(), lead.ID, fx.tenantID, "primeira nota", creator)
	require.NoError(t, err)
	require.NoError(t, repos.notes.Create(ctx, n1))

	// Delay > 1s so ORDER BY created_at DESC is deterministic
	// (created_at is DATETIME with second-level precision).
	time.Sleep(1100 * time.Millisecond)

	n2, err := domain.NewLeadNote(uuid.New().String(), lead.ID, fx.tenantID, "segunda nota", creator)
	require.NoError(t, err)
	require.NoError(t, repos.notes.Create(ctx, n2))

	list, err := repos.notes.FindByLeadID(ctx, lead.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// ORDER BY created_at DESC — newest first.
	assert.Equal(t, n2.ID, list[0].ID)
	assert.Equal(t, "segunda nota", list[0].Content)
	assert.Equal(t, n1.ID, list[1].ID)
}

func TestGormLeadNoteRepository_FindByLeadID_Empty(t *testing.T) {
	repos, _ := setupFunnelRepos(t)

	list, err := repos.notes.FindByLeadID(context.Background(), uuid.New().String())
	require.NoError(t, err)
	assert.Empty(t, list)
}
