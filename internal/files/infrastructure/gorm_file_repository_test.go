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

	"github.com/sasrgita/crm-juridico/internal/files/domain"
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

func setupFileRepo(t *testing.T) (*GormFileRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	// Order matters due to FK constraints.
	db.Exec("DELETE FROM files")
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

	return NewGormFileRepository(db), db
}

func createTenant(t *testing.T, db *gorm.DB) string {
	t.Helper()
	repo := tenantInfra.NewGormTenantRepository(db)
	tn, err := tenantDomain.NewTenant(uuid.New().String(), "Escritorio Teste", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), tn))
	return tn.ID
}

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

func createConversation(t *testing.T, db *gorm.DB, tenantID, contactID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO conversations (id, tenant_id, contact_id, status, last_message_at, unread_count, created_at, updated_at)
		VALUES (?, ?, ?, 'open', NOW(), 0, NOW(), NOW())`,
		id, tenantID, contactID).Error
	require.NoError(t, err)
	return id
}

// mkFile builds a valid domain.File with fresh UUIDs.
func mkFile(t *testing.T, tenantID, convID, contactID, name string, mt domain.MediaType) *domain.File {
	t.Helper()
	f, err := domain.NewFile(
		uuid.New().String(), tenantID, convID, contactID,
		name, "application/octet-stream",
		"k/"+uuid.New().String(),
		int64(len(name)),
		mt, domain.DirectionInbound, nil, nil,
	)
	require.NoError(t, err)
	return f
}

func TestGormFileRepository_Create_And_FindByID(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+55119")
	convID := createConversation(t, db, tenantID, contactID)

	f := mkFile(t, tenantID, convID, contactID, "contrato.pdf", domain.MediaTypeDocument)
	require.NoError(t, repo.Create(ctx, f))

	got, err := repo.FindByID(ctx, tenantID, f.ID)
	require.NoError(t, err)
	assert.Equal(t, f.ID, got.ID)
	assert.Equal(t, tenantID, got.TenantID)
	assert.Equal(t, "contrato.pdf", got.Name)
	assert.Equal(t, domain.MediaTypeDocument, got.MediaType)
	assert.Equal(t, domain.DirectionInbound, got.Direction)
}

func TestGormFileRepository_FindByID_NotFound(t *testing.T) {
	repo, db := setupFileRepo(t)
	tenantID := createTenant(t, db)
	_, err := repo.FindByID(context.Background(), tenantID, uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrFileNotFound)
}

func TestGormFileRepository_FindByID_CrossTenantReturnsNotFound(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tA := createTenant(t, db)
	tB := createTenant(t, db)
	cA := createContact(t, db, tA, "A", "+1")
	convA := createConversation(t, db, tA, cA)

	fA := mkFile(t, tA, convA, cA, "x.pdf", domain.MediaTypeDocument)
	require.NoError(t, repo.Create(ctx, fA))

	_, err := repo.FindByID(ctx, tB, fA.ID)
	assert.ErrorIs(t, err, domain.ErrFileNotFound)
}

func TestGormFileRepository_List_OrderAndPagination(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+55119")
	convID := createConversation(t, db, tenantID, contactID)

	// Insert 3 files with increasing created_at.
	for i, name := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		f := mkFile(t, tenantID, convID, contactID, name, domain.MediaTypeDocument)
		f.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		f.UpdatedAt = f.CreatedAt
		require.NoError(t, repo.Create(ctx, f))
	}

	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, PageSize: 2, Page: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.Total)
	require.Len(t, res.Items, 2)
	assert.Equal(t, "c.pdf", res.Items[0].Name)
	assert.Equal(t, "b.pdf", res.Items[1].Name)
	assert.Equal(t, "Maria", res.Items[0].ContactName)

	res2, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, PageSize: 2, Page: 2})
	require.NoError(t, err)
	require.Len(t, res2.Items, 1)
	assert.Equal(t, "a.pdf", res2.Items[0].Name)
}

func TestGormFileRepository_List_TenantIsolation(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tA := createTenant(t, db)
	tB := createTenant(t, db)
	cA := createContact(t, db, tA, "A", "+1")
	cB := createContact(t, db, tB, "B", "+2")
	convA := createConversation(t, db, tA, cA)
	convB := createConversation(t, db, tB, cB)

	require.NoError(t, repo.Create(ctx, mkFile(t, tA, convA, cA, "a.pdf", domain.MediaTypeDocument)))
	require.NoError(t, repo.Create(ctx, mkFile(t, tB, convB, cB, "b.pdf", domain.MediaTypeDocument)))

	res, err := repo.List(ctx, domain.ListQuery{TenantID: tA})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "a.pdf", res.Items[0].Name)
}

func TestGormFileRepository_List_FilterByMediaType(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+1")
	convID := createConversation(t, db, tenantID, contactID)

	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "img.jpg", domain.MediaTypeImage)))
	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "doc.pdf", domain.MediaTypeDocument)))

	mt := domain.MediaTypeImage
	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, MediaType: &mt})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "img.jpg", res.Items[0].Name)
}

func TestGormFileRepository_List_SearchByName(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+1")
	convID := createConversation(t, db, tenantID, contactID)

	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "contrato-A.pdf", domain.MediaTypeDocument)))
	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "Contrato-B.pdf", domain.MediaTypeDocument)))
	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "foto.jpg", domain.MediaTypeImage)))

	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, Search: "contrato"})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
}

func TestGormFileRepository_List_SearchEscapesLikeWildcards(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+1")
	convID := createConversation(t, db, tenantID, contactID)

	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "100_percent.txt", domain.MediaTypeDocument)))
	require.NoError(t, repo.Create(ctx, mkFile(t, tenantID, convID, contactID, "other.txt", domain.MediaTypeDocument)))

	// Literal "%" should not match everything; should only match names containing "%".
	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, Search: "%"})
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.Total, "unescaped %% would match all rows")
}

func TestGormFileRepository_List_DateRange(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+1")
	convID := createConversation(t, db, tenantID, contactID)

	now := time.Now().UTC()
	for i, name := range []string{"old.pdf", "middle.pdf", "new.pdf"} {
		f := mkFile(t, tenantID, convID, contactID, name, domain.MediaTypeDocument)
		f.CreatedAt = now.Add(time.Duration(-(2 - i)) * 24 * time.Hour)
		f.UpdatedAt = f.CreatedAt
		require.NoError(t, repo.Create(ctx, f))
	}

	from := now.Add(-36 * time.Hour)
	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, From: &from})
	require.NoError(t, err)
	// Expect middle + new (last 36h)
	assert.Equal(t, int64(2), res.Total)
}

func TestGormFileRepository_CountAndListRecentByLead(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	contactID := createContact(t, db, tenantID, "Maria", "+1")
	convID := createConversation(t, db, tenantID, contactID)

	// Create a funnel + column + lead via raw SQL to avoid depending on the
	// funnel infrastructure package here.
	funnelID := uuid.New().String()
	require.NoError(t, db.Exec(`INSERT INTO funnels (id, tenant_id, name, description, active, is_default, created_at, updated_at)
		VALUES (?, ?, 'F', '', 1, 1, NOW(), NOW())`, funnelID, tenantID).Error)
	columnID := uuid.New().String()
	require.NoError(t, db.Exec(`INSERT INTO funnel_columns (id, funnel_id, name, order_index, type, color, created_at, updated_at)
		VALUES (?, ?, 'Novo', 0, 'entry', '#22c55e', NOW(), NOW())`, columnID, funnelID).Error)
	leadID := uuid.New().String()
	require.NoError(t, db.Exec(`INSERT INTO leads (id, tenant_id, funnel_id, column_id, contact_id, conversation_id, score, status, column_entered_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 'open', NOW(), NOW(), NOW())`,
		leadID, tenantID, funnelID, columnID, contactID, convID).Error)

	for i := 0; i < 8; i++ {
		f := mkFile(t, tenantID, convID, contactID, "f.pdf", domain.MediaTypeDocument)
		f.LeadID = &leadID
		f.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		f.UpdatedAt = f.CreatedAt
		require.NoError(t, repo.Create(ctx, f))
	}

	count, err := repo.CountByLead(ctx, tenantID, leadID)
	require.NoError(t, err)
	assert.Equal(t, int64(8), count)

	recent, err := repo.ListRecentByLead(ctx, tenantID, leadID, 6)
	require.NoError(t, err)
	assert.Len(t, recent, 6)
}

func TestGormFileRepository_List_PageSizeCaps(t *testing.T) {
	repo, db := setupFileRepo(t)
	ctx := context.Background()

	tenantID := createTenant(t, db)
	res, err := repo.List(ctx, domain.ListQuery{TenantID: tenantID, PageSize: 1000})
	require.NoError(t, err)
	assert.LessOrEqual(t, res.PageSize, domain.MaxPageSize)
}

func TestGormFileRepository_List_EmptyTenantRejected(t *testing.T) {
	repo, _ := setupFileRepo(t)
	_, err := repo.List(context.Background(), domain.ListQuery{TenantID: ""})
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}
