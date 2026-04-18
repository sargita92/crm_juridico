package notification_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/notification"
	"github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
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

// setupDB returns a clean DB with schema migrated. Tables referenced by the
// notification FK constraints (tenants, users) are emptied before the test.
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	// Clean tables the notification test touches. Order respects FKs.
	db.Exec("DELETE FROM notifications")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	return db
}

// insertTenant inserts a minimal tenant row and returns its ID.
func insertTenant(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO tenants (id, name, type, document, status, created_at, updated_at)
		VALUES (?, ?, 'PJ', ?, 'active', NOW(), NOW())`,
		id, "Escritorio Teste "+id[:8], id[:14]).Error
	require.NoError(t, err)
	return id
}

// insertUser inserts a minimal user row and returns its ID.
func insertUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(`INSERT INTO users (id, name, email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, '$2a$10$fakehash1234567890123456789012345678901234567890', 'user', 'active', NOW(), NOW())`,
		id, "User "+id[:8], id[:8]+"@test.com").Error
	require.NoError(t, err)
	return id
}

func TestNotificationModule_CreatesNotification_OnResponsibleAssigned(t *testing.T) {
	db := setupDB(t)
	tenantID := insertTenant(t, db)
	userID := insertUser(t, db)

	bus := events.NewMemoryEventBus()
	mod := notification.NewModule(db, bus, zap.NewNop())
	require.NotNil(t, mod)

	// Give the subscriber goroutine a moment to register with the bus before
	// we publish, so the event isn't lost.
	time.Sleep(20 * time.Millisecond)

	bus.Publish(events.Event{
		Type:     events.EventLeadResponsibleAssigned,
		TenantID: tenantID,
		Payload: events.ResponsibleAssignedPayload{
			LeadID:            "lead-1",
			ResponsibleUserID: userID,
			Reason:            "created",
			Outcome:           "picked",
			Algorithm:         "round_robin",
		},
	})

	// Poll up to 2s for the notification to appear; goroutine runs async.
	repo := infrastructure.NewGormNotificationRepository(db)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		found, err := repo.FindByUserID(context.Background(), tenantID, userID, false, 10, 0)
		require.NoError(t, err)
		if len(found) > 0 {
			require.Len(t, found, 1)
			n := found[0]
			assert.Equal(t, tenantID, n.TenantID)
			assert.Equal(t, userID, n.UserID)
			assert.Equal(t, "lead_assigned", string(n.Type))
			assert.Equal(t, "Novo lead atribuído", n.Title)
			assert.Equal(t, "Você recebeu um novo lead.", n.Body)
			assert.Equal(t, "lead-1", n.Metadata["lead_id"])
			assert.Equal(t, "created", n.Metadata["reason"])
			assert.Equal(t, "picked", n.Metadata["outcome"])
			assert.Equal(t, "round_robin", n.Metadata["algorithm"])
			assert.False(t, n.Read)
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for notification to be persisted")
}

func TestNotificationModule_IgnoresOtherEventTypes(t *testing.T) {
	db := setupDB(t)
	tenantID := insertTenant(t, db)
	userID := insertUser(t, db)

	bus := events.NewMemoryEventBus()
	mod := notification.NewModule(db, bus, zap.NewNop())
	require.NotNil(t, mod)

	time.Sleep(20 * time.Millisecond)

	// Publish an unrelated event type; listener must not create a notification.
	bus.Publish(events.Event{
		Type:     events.EventLeadCreated,
		TenantID: tenantID,
		Payload:  map[string]string{"lead_id": "lead-xyz"},
	})

	// Wait briefly; we expect zero notifications.
	time.Sleep(200 * time.Millisecond)

	repo := infrastructure.NewGormNotificationRepository(db)
	found, err := repo.FindByUserID(context.Background(), tenantID, userID, false, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestNotificationModule_IgnoresMalformedPayload(t *testing.T) {
	db := setupDB(t)
	tenantID := insertTenant(t, db)
	userID := insertUser(t, db)

	bus := events.NewMemoryEventBus()
	mod := notification.NewModule(db, bus, zap.NewNop())
	require.NotNil(t, mod)

	time.Sleep(20 * time.Millisecond)

	// Correct type but wrong payload shape — listener should log Warn and continue.
	bus.Publish(events.Event{
		Type:     events.EventLeadResponsibleAssigned,
		TenantID: tenantID,
		Payload:  "not-a-struct",
	})

	time.Sleep(200 * time.Millisecond)

	repo := infrastructure.NewGormNotificationRepository(db)
	found, err := repo.FindByUserID(context.Background(), tenantID, userID, false, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, found)
}
