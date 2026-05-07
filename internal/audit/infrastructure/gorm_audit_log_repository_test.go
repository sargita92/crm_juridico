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

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
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

func setupAuditRepo(t *testing.T) (*GormAuditLogRepository, *gorm.DB) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	// Limpa a tabela para isolamento.
	require.NoError(t, db.Exec("DELETE FROM audit_logs").Error)

	return NewGormAuditLogRepository(db), db
}

func ptrStr(s string) *string { return &s }

func mustNewLog(t *testing.T, in auditdomain.NewAuditLogInput) *auditdomain.AuditLog {
	t.Helper()
	if in.ActorEmail == "" {
		in.ActorEmail = "admin@example.com"
	}
	if in.Action == "" {
		in.Action = auditdomain.ActionLoginSuccess
	}
	if in.Entity == "" {
		in.Entity = "session"
	}
	if in.IP == "" {
		in.IP = "127.0.0.1"
	}
	log, err := auditdomain.NewAuditLog(in)
	require.NoError(t, err)
	return log
}

func TestGormAuditLogRepository_Create_AllFields(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	tenantID := uuid.NewString()
	userID := uuid.NewString()
	entityID := uuid.NewString()
	log := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(tenantID),
		UserID:     ptrStr(userID),
		ActorEmail: "admin@example.com",
		Action:     auditdomain.ActionTenantCreated,
		Entity:     "tenant",
		EntityID:   ptrStr(entityID),
		IP:         "203.0.113.42",
		UserAgent:  "Mozilla/5.0",
		Metadata:   auditdomain.Metadata{"name": "Escritorio Silva", "type": "PJ"},
	})

	require.NoError(t, repo.Create(ctx, log))

	found, err := repo.FindByID(ctx, log.ID)
	require.NoError(t, err)
	assert.Equal(t, log.ID, found.ID)
	require.NotNil(t, found.TenantID)
	assert.Equal(t, tenantID, *found.TenantID)
	require.NotNil(t, found.UserID)
	assert.Equal(t, userID, *found.UserID)
	assert.Equal(t, "admin@example.com", found.ActorEmail)
	assert.Equal(t, auditdomain.ActionTenantCreated, found.Action)
	assert.Equal(t, "tenant", found.Entity)
	require.NotNil(t, found.EntityID)
	assert.Equal(t, entityID, *found.EntityID)
	assert.Equal(t, "203.0.113.42", found.IP)
	assert.Equal(t, "Mozilla/5.0", found.UserAgent)
	assert.Equal(t, "Escritorio Silva", found.Metadata["name"])
	assert.Equal(t, "PJ", found.Metadata["type"])
	// CreatedAt UTC preservado (ate o milissegundo, dentro da precisao DATETIME(3)).
	assert.Equal(t, time.UTC, found.CreatedAt.Location())
	assert.WithinDuration(t, log.CreatedAt, found.CreatedAt, 2*time.Millisecond)
}

func TestGormAuditLogRepository_Create_NullableTenantAndUser(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	// Falha de login: TenantID e UserID nil, ActorEmail preenchido com tentativa.
	log := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "ghost@example.com",
		Action:     auditdomain.ActionLoginFailure,
		Entity:     "session",
		IP:         "203.0.113.50",
	})
	require.NoError(t, repo.Create(ctx, log))

	found, err := repo.FindByID(ctx, log.ID)
	require.NoError(t, err)
	assert.Nil(t, found.TenantID)
	assert.Nil(t, found.UserID)
	assert.Equal(t, "ghost@example.com", found.ActorEmail)
}

func TestGormAuditLogRepository_Create_EmptyMetadata(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	log := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "admin@example.com",
		Action:     auditdomain.ActionLogout,
		Entity:     "session",
		IP:         "127.0.0.1",
		// Metadata nil de proposition.
	})
	require.NoError(t, repo.Create(ctx, log))

	found, err := repo.FindByID(ctx, log.ID)
	require.NoError(t, err)
	// Metadata vazia/nil retorna como mapa nao-nil mas vazio (ou nil).
	assert.Empty(t, found.Metadata)
}

func TestGormAuditLogRepository_FindByID_NotFound(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, uuid.NewString())
	assert.ErrorIs(t, err, auditdomain.ErrAuditLogNotFound)
}

func TestGormAuditLogRepository_List_OrderingDesc(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	older := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
		CreatedAt:  now.Add(-2 * time.Hour),
	})
	mid := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "b@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
		CreatedAt:  now.Add(-1 * time.Hour),
	})
	newest := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "c@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
		CreatedAt:  now,
	})

	require.NoError(t, repo.Create(ctx, older))
	require.NoError(t, repo.Create(ctx, mid))
	require.NoError(t, repo.Create(ctx, newest))

	logs, total, err := repo.List(ctx, auditdomain.Filter{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, logs, 3)
	assert.Equal(t, newest.ID, logs[0].ID)
	assert.Equal(t, mid.ID, logs[1].ID)
	assert.Equal(t, older.ID, logs[2].ID)
}

func TestGormAuditLogRepository_List_Pagination(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := range 25 {
		log := mustNewLog(t, auditdomain.NewAuditLogInput{
			ActorEmail: "a@example.com",
			Action:     auditdomain.ActionLoginSuccess,
			Entity:     "session",
			IP:         "127.0.0.1",
			CreatedAt:  now.Add(time.Duration(-i) * time.Second),
		})
		require.NoError(t, repo.Create(ctx, log))
	}

	page1, total, err := repo.List(ctx, auditdomain.Filter{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(25), total)
	assert.Len(t, page1, 10)

	page2, _, err := repo.List(ctx, auditdomain.Filter{Page: 2, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, page2, 10)

	page3, _, err := repo.List(ctx, auditdomain.Filter{Page: 3, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, page3, 5)

	// Garante que paginas nao se sobrepoem.
	seen := map[string]bool{}
	for _, p := range [][]*auditdomain.AuditLog{page1, page2, page3} {
		for _, l := range p {
			assert.False(t, seen[l.ID], "id %s repetiu entre paginas", l.ID)
			seen[l.ID] = true
		}
	}
	assert.Len(t, seen, 25)
}

func TestGormAuditLogRepository_List_FilterByTenant(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	tenantA := uuid.NewString()
	tenantB := uuid.NewString()

	logA := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(tenantA),
		ActorEmail: "x@example.com",
		Action:     auditdomain.ActionTenantUpdated,
		Entity:     "tenant",
		EntityID:   ptrStr(tenantA),
		IP:         "127.0.0.1",
	})
	logB := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(tenantB),
		ActorEmail: "y@example.com",
		Action:     auditdomain.ActionTenantUpdated,
		Entity:     "tenant",
		EntityID:   ptrStr(tenantB),
		IP:         "127.0.0.1",
	})
	logNullTenant := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "z@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
	})

	require.NoError(t, repo.Create(ctx, logA))
	require.NoError(t, repo.Create(ctx, logB))
	require.NoError(t, repo.Create(ctx, logNullTenant))

	logs, total, err := repo.List(ctx, auditdomain.Filter{TenantID: ptrStr(tenantA)})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, logA.ID, logs[0].ID)
}

func TestGormAuditLogRepository_List_FilterByUser(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	userA := uuid.NewString()
	userB := uuid.NewString()

	la := mustNewLog(t, auditdomain.NewAuditLogInput{
		UserID:     ptrStr(userA),
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "1.1.1.1",
	})
	lb := mustNewLog(t, auditdomain.NewAuditLogInput{
		UserID:     ptrStr(userB),
		ActorEmail: "b@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "1.1.1.1",
	})
	require.NoError(t, repo.Create(ctx, la))
	require.NoError(t, repo.Create(ctx, lb))

	logs, total, err := repo.List(ctx, auditdomain.Filter{UserID: ptrStr(userA)})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, la.ID, logs[0].ID)
}

func TestGormAuditLogRepository_List_FilterByAction(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	login := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "1.1.1.1",
	})
	logout := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionLogout,
		Entity:     "session",
		IP:         "1.1.1.1",
	})
	require.NoError(t, repo.Create(ctx, login))
	require.NoError(t, repo.Create(ctx, logout))

	action := auditdomain.ActionLogout
	logs, total, err := repo.List(ctx, auditdomain.Filter{Action: &action})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, logout.ID, logs[0].ID)
}

func TestGormAuditLogRepository_List_FilterByPeriod(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	now := time.Now().UTC()
	old := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "old@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "1.1.1.1",
		CreatedAt:  now.Add(-72 * time.Hour),
	})
	recent := mustNewLog(t, auditdomain.NewAuditLogInput{
		ActorEmail: "recent@example.com",
		Action:     auditdomain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "1.1.1.1",
		CreatedAt:  now.Add(-1 * time.Hour),
	})
	require.NoError(t, repo.Create(ctx, old))
	require.NoError(t, repo.Create(ctx, recent))

	from := now.Add(-24 * time.Hour)
	to := now.Add(time.Minute)
	logs, total, err := repo.List(ctx, auditdomain.Filter{From: &from, To: &to})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, recent.ID, logs[0].ID)
}

func TestGormAuditLogRepository_List_CombinedFilters(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	tenantA := uuid.NewString()
	target := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(tenantA),
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionTenantDeactivated,
		Entity:     "tenant",
		EntityID:   ptrStr(tenantA),
		IP:         "1.1.1.1",
	})
	noiseSameTenant := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(tenantA),
		ActorEmail: "a@example.com",
		Action:     auditdomain.ActionTenantUpdated,
		Entity:     "tenant",
		EntityID:   ptrStr(tenantA),
		IP:         "1.1.1.1",
	})
	noiseSameAction := mustNewLog(t, auditdomain.NewAuditLogInput{
		TenantID:   ptrStr(uuid.NewString()),
		ActorEmail: "x@example.com",
		Action:     auditdomain.ActionTenantDeactivated,
		Entity:     "tenant",
		EntityID:   ptrStr(uuid.NewString()),
		IP:         "1.1.1.1",
	})
	require.NoError(t, repo.Create(ctx, target))
	require.NoError(t, repo.Create(ctx, noiseSameTenant))
	require.NoError(t, repo.Create(ctx, noiseSameAction))

	action := auditdomain.ActionTenantDeactivated
	logs, total, err := repo.List(ctx, auditdomain.Filter{
		TenantID: ptrStr(tenantA),
		Action:   &action,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, target.ID, logs[0].ID)
}

func TestGormAuditLogRepository_List_EmptyResultsAndOutOfRangePage(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	for i := range 3 {
		log := mustNewLog(t, auditdomain.NewAuditLogInput{
			ActorEmail: "a@example.com",
			Action:     auditdomain.ActionLoginSuccess,
			Entity:     "session",
			IP:         "1.1.1.1",
			CreatedAt:  time.Now().UTC().Add(time.Duration(-i) * time.Second),
		})
		require.NoError(t, repo.Create(ctx, log))
	}

	// Pagina fora do range: total correto, len = 0.
	logs, total, err := repo.List(ctx, auditdomain.Filter{Page: 5, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Empty(t, logs)

	// Filtro que nao casa: total 0 e len 0.
	tenantNotPresent := uuid.NewString()
	logs2, total2, err := repo.List(ctx, auditdomain.Filter{TenantID: ptrStr(tenantNotPresent)})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total2)
	assert.Empty(t, logs2)
}

func TestGormAuditLogRepository_List_InvalidFilter_Propagates(t *testing.T) {
	repo, _ := setupAuditRepo(t)
	ctx := context.Background()

	from := time.Now().UTC()
	to := from.Add(-time.Hour)
	_, _, err := repo.List(ctx, auditdomain.Filter{From: &from, To: &to})
	assert.ErrorIs(t, err, auditdomain.ErrInvalidPeriod)
}
