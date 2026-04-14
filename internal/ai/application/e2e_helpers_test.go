//go:build integration

package application_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/ai"
	aidomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"

	// Module constructors (test package, so importing the parent `ai` package
	// and its siblings is fine — there is no import cycle).
	"github.com/sasrgita/crm-juridico/internal/document"
	"github.com/sasrgita/crm-juridico/internal/funnel"
	funnelinfra "github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/product"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/specialist"
	"github.com/sasrgita/crm-juridico/internal/tenant"
	"github.com/sasrgita/crm-juridico/internal/whatsapp"
	whatsappapp "github.com/sasrgita/crm-juridico/internal/whatsapp/application"
)

// -----------------------------------------------------------------------------
// Shared MySQL container for all integration tests in this package.
// Follows the pattern from internal/shared/database/database_test.go.
// -----------------------------------------------------------------------------

var (
	sharedContainer     *testhelper.MySQLContainer
	sharedContainerOnce sync.Once
	sharedContainerErr  error
	fixturesLoadedOnce  sync.Once
	fixturesLoadErr     error
	migrationsRunOnce   sync.Once
	migrationsRunErr    error
)

func ensureSharedContainer(t *testing.T) *testhelper.MySQLContainer {
	t.Helper()
	sharedContainerOnce.Do(func() {
		ctx := context.Background()
		defer func() {
			if r := recover(); r != nil {
				sharedContainerErr = &panicErr{msg: "panic starting mysql container"}
			}
		}()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
	})
	if sharedContainerErr != nil {
		t.Fatalf("failed to start shared mysql container: %v", sharedContainerErr)
	}
	return sharedContainer
}

type panicErr struct{ msg string }

func (p *panicErr) Error() string { return p.msg }

// -----------------------------------------------------------------------------
// fakeWhatsAppProvider: minimal implementation of whatsappdomain.WhatsAppProvider
// used by the integration harness. SendMessage is a no-op — the engine still
// persists outgoing messages via SendMessageUseCase which writes them to the
// DB, we just don't want to actually touch whatsmeow.
// -----------------------------------------------------------------------------

type fakeWhatsAppProvider struct{}

func (f *fakeWhatsAppProvider) Connect(ctx context.Context, tenantID string) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}
func (f *fakeWhatsAppProvider) Disconnect(ctx context.Context, tenantID string) error { return nil }
func (f *fakeWhatsAppProvider) IsConnected(tenantID string) bool                      { return true }
func (f *fakeWhatsAppProvider) SendMessage(ctx context.Context, tenantID, recipientWhatsAppID, content string) (string, error) {
	return "fake-" + uuid.New().String(), nil
}
func (f *fakeWhatsAppProvider) SetMessageHandler(handler whatsappdomain.IncomingMessageHandler) {}

// -----------------------------------------------------------------------------
// testEnv: wires the full AI pipeline against the shared MySQL container.
// -----------------------------------------------------------------------------

type testEnvOpts struct {
	ResetCommandEnabled bool
}

type testEnvOpt func(*testEnvOpts)

type testEnv struct {
	t                *testing.T
	db               *gorm.DB
	aiMod            *ai.Module
	contactRepo      whatsappdomain.ContactRepository
	conversationRepo whatsappdomain.ConversationRepository
	messageRepo      whatsappdomain.MessageRepository
	leadRepo         funneldomain.LeadRepository
	stateRepo        aidomain.ConversationStateRepository
	receiveUC        *whatsappapp.ReceiveMessageUseCase
}

func setupTestEnv(t *testing.T, opts ...testEnvOpt) *testEnv {
	t.Helper()

	o := testEnvOpts{ResetCommandEnabled: true}
	for _, fn := range opts {
		fn(&o)
	}

	container := ensureSharedContainer(t)
	db := container.DB(t)
	log := container.Logger()

	// Run migrations exactly once for the entire package run.
	migrationsRunOnce.Do(func() {
		migrationsRunErr = database.RunMigrations(db, log, migrationsSourceURL())
	})
	if migrationsRunErr != nil {
		t.Fatalf("migrations: %v", migrationsRunErr)
	}

	// Load fixtures exactly once (the fixture SQL uses session-scoped SET @var
	// statements which are safe to run multiple times via multiStatements, but
	// they have been observed to hit odd FK races on re-load — loading once
	// keeps the harness deterministic).
	fixturesLoadedOnce.Do(func() {
		fixturesLoadErr = doLoadFixtures(db)
	})
	if fixturesLoadErr != nil {
		t.Fatalf("loadFixtures: %v", fixturesLoadErr)
	}

	// Wipe the F17 playground contact (and all its artifacts) so that each
	// test starts from a known-clean slate. The fixture pre-seeds the contact
	// + conversation but no lead; we delete them so a fresh conversation is
	// created by ReceiveMessageUC, which fires the LeadCreator — we need a
	// lead to make assertions against.
	cleanupPlaygroundContact(t, db)

	// --- Wire modules (mirrors cmd/api/main.go). ---
	tenantMod := tenant.NewModule(db)
	specialistMod := specialist.NewModule(db, tenantMod.TenantRepo())
	documentMod := document.NewModule(db, specialistMod.SpecialistRepo())

	whatsmeowProvider := &fakeWhatsAppProvider{}
	eventBus := events.NewMemoryEventBus()
	whatsappMod := whatsapp.NewModule(db, whatsmeowProvider, eventBus, log)

	productMod := product.NewModule(db, log)

	// Cross-module adapters for funnel.
	contactAdapter := funnelinfra.NewWhatsAppContactAdapter(whatsappMod.ContactRepo())
	messageAdapter := funnelinfra.NewWhatsAppMessageAdapter(whatsappMod.MessageRepo())
	// The UserNameAdapter expects a user repository; for tests we can reuse
	// the one provided by the auth infrastructure layer via the adapter's
	// constructor. Since funnel's ListFunnels use case doesn't depend on it
	// for our flow, we pass nil and rely on the fact that CreateLead doesn't
	// touch user names.
	userNameAdapter := funnelinfra.NewUserNameAdapter(nil)
	productDetectorAdapter := funnelinfra.NewProductDetectorAdapter(productMod.DetectUseCase())
	productProviderAdapter := funnelinfra.NewProductProviderAdapter(productMod.ProductRepo())
	funnelProductRouterAdapter := funnelinfra.NewFunnelProductRouterAdapter(productMod.FunnelProductRepo())
	productListerAdapter := funnelinfra.NewProductListerAdapter(productMod.ProductRepo(), productMod.TenantProductRepo())

	funnelMod := funnel.NewModule(
		db,
		contactAdapter,
		messageAdapter,
		userNameAdapter,
		productDetectorAdapter,
		productProviderAdapter,
		funnelProductRouterAdapter,
		productListerAdapter,
		eventBus,
		log,
	)
	whatsappMod.SetLeadCreator(funnelMod.LeadCreator())

	// AI module wiring — uses cfg.DefaultDebounce=1s so tests don't wait long.
	aiCfg := config.AIConfigEnv{
		DefaultProvider:     "fake",
		DefaultModel:        "fake-v1",
		DefaultMaxTokens:    1024,
		DefaultTemperature:  0.7,
		DefaultDebounce:     1,
		PlaygroundEnabled:   false,
		ResetCommandEnabled: o.ResetCommandEnabled,
	}

	aiMod := ai.NewModule(db, aiCfg, log, ai.ModuleDeps{
		SpecialistRepo:   specialistMod.SpecialistRepo(),
		StepRepo:         specialistMod.StepRepo(),
		GuardrailRepo:    specialistMod.GuardrailRepo(),
		SpecTenantRepo:   specialistMod.SpecTenantRepo(),
		DocRepo:          documentMod.DocRepo(),
		SpecDocRepo:      documentMod.SpecDocRepo(),
		ProductRepo:      productMod.ProductRepo(),
		PhoneNumberRepo:  productMod.PhoneNumberRepo(),
		DetectProductUC:  productMod.DetectUseCase(),
		MessageRepo:      whatsappMod.MessageRepo(),
		ConversationRepo: whatsappMod.ConversationRepo(),
		SendMessageUC:    whatsappMod.SendMessageUC(),
		ReceiveMessageUC: whatsappMod.ReceiveMessageUC(),
		LeadRepo:         funnelMod.LeadRepo(),
		MoveLeadUC:       funnelMod.MoveLeadUC(),
		FunnelRepo:       funnelMod.FunnelRepo(),
		ColumnRepo:       funnelMod.ColumnRepo(),
	})
	whatsappMod.SetAIHandler(aiMod)

	env := &testEnv{
		t:                t,
		db:               db,
		aiMod:            aiMod,
		contactRepo:      whatsappMod.ContactRepo(),
		conversationRepo: whatsappMod.ConversationRepo(),
		messageRepo:      whatsappMod.MessageRepo(),
		leadRepo:         funnelMod.LeadRepo(),
		stateRepo:        aiMod.StateRepo(),
		receiveUC:        whatsappMod.ReceiveMessageUC(),
	}

	return env
}

// Close is a no-op; the shared container lives for the duration of the test
// process. Kept here so tests can call `defer env.Close()` idiomatically.
func (e *testEnv) Close() {}

// SendInbound simulates an incoming WhatsApp message by calling the real
// ReceiveMessageUseCase, which in turn triggers the AI handler goroutine.
func (e *testEnv) SendInbound(t *testing.T, tenantID, phone, jid, content string) {
	t.Helper()
	err := e.receiveUC.Execute(context.Background(), whatsappdomain.IncomingMessage{
		TenantID:      tenantID,
		SenderJID:     jid,
		SenderName:    "Teste E2E",
		SenderPhone:   phone,
		Content:       content,
		WhatsAppMsgID: "wamid.test-" + uuid.New().String(),
		Timestamp:     time.Now(),
	})
	require.NoError(t, err, "ReceiveMessageUC.Execute")
}

// AwaitDebouncer sleeps long enough for the debouncer window (configured to
// 1s in the harness) to fire, plus a small buffer for the engine to persist.
func (e *testEnv) AwaitDebouncer(t *testing.T, d time.Duration) {
	t.Helper()
	time.Sleep(d + 300*time.Millisecond)
}

// MustLoadState loads the ConversationState for the given conversation ID.
func (e *testEnv) MustLoadState(t *testing.T, conversationID string) *aidomain.ConversationState {
	t.Helper()
	state, err := e.stateRepo.FindByConversationID(context.Background(), conversationID)
	require.NoError(t, err, "stateRepo.FindByConversationID")
	require.NotNil(t, state, "state must not be nil")
	return state
}

// MustLoadLead loads the Lead for the given conversation ID.
func (e *testEnv) MustLoadLead(t *testing.T, conversationID string) *funneldomain.Lead {
	t.Helper()
	lead, err := e.leadRepo.FindByConversationID(context.Background(), conversationID)
	require.NoError(t, err, "leadRepo.FindByConversationID")
	require.NotNil(t, lead, "lead must not be nil")
	return lead
}

// ListMessages returns all messages for a conversation, oldest first.
func (e *testEnv) ListMessages(t *testing.T, tenantID, conversationID string) []whatsappdomain.Message {
	t.Helper()
	msgs, err := e.messageRepo.FindByConversationID(context.Background(), conversationID, whatsappdomain.MessageFilter{Limit: 100})
	require.NoError(t, err, "messageRepo.FindByConversationID")
	return msgs
}

// SetStepIndex fast-forwards the conversation state to a given step. This is
// a test-only escape hatch for walking past free_text steps that the rule
// evaluator cannot satisfy without an LLM. It updates the row directly via
// SQL so that it bypasses any GORM quirks around saving a row whose PK is
// still the engine's default empty string.
func (e *testEnv) SetStepIndex(t *testing.T, conversationID string, stepIndex int) {
	t.Helper()
	sqlDB, err := e.db.DB()
	require.NoError(t, err)
	_, err = sqlDB.ExecContext(context.Background(),
		"UPDATE conversation_states SET current_step_index = ? WHERE conversation_id = ?",
		stepIndex, conversationID,
	)
	require.NoError(t, err, "SetStepIndex UPDATE")
}

// MustConversationIDForPhone resolves a conversation ID from a phone number.
func (e *testEnv) MustConversationIDForPhone(t *testing.T, tenantID, phone string) string {
	t.Helper()
	jid := phone + "@s.whatsapp.net"
	contact, err := e.contactRepo.FindByWhatsAppID(context.Background(), tenantID, jid)
	require.NoError(t, err, "contactRepo.FindByWhatsAppID")
	require.NotNil(t, contact)
	conv, err := e.conversationRepo.FindByContactID(context.Background(), tenantID, contact.ID)
	require.NoError(t, err, "conversationRepo.FindByContactID")
	require.NotNil(t, conv)
	return conv.ID
}

// -----------------------------------------------------------------------------
// Fixture loading + cleanup helpers.
// -----------------------------------------------------------------------------

func migrationsSourceURL() string {
	_, filename, _, _ := runtime.Caller(0)
	// .../internal/ai/application/e2e_helpers_test.go → project root is 4 levels up.
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return "file://" + filepath.Join(root, "migrations")
}

func fixturePath() string {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	return filepath.Join(root, "fixture", "fixtures.sql")
}

// doLoadFixtures reads the fixture SQL and executes it in one round-trip
// using the underlying *sql.DB. The DSN sets multiStatements=true so the
// whole file executes in a single call (session variables like @CONTACT_TESTE
// stay bound to that connection for the duration of the multi-statement
// query).
func doLoadFixtures(db *gorm.DB) error {
	sqlBytes, err := os.ReadFile(fixturePath())
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	if _, err := sqlDB.Exec(string(sqlBytes)); err != nil {
		return err
	}
	return nil
}

// cleanupPlaygroundContact deletes the F17 playground contact (and anything
// referring to it) so that each integration test can start from a clean state.
// We don't touch the rest of the fixture data (specialist, steps, products, …).
func cleanupPlaygroundContact(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)

	phone := "5511900000000"
	ctx := context.Background()
	// Delete in dependency order. Foreign keys should be set to CASCADE for
	// most of these, but we delete explicitly to be safe across environments.
	statements := []string{
		// Lead + movements via the contact → conversation chain.
		`DELETE m FROM lead_movements m
		 INNER JOIN leads l ON l.id = m.lead_id
		 INNER JOIN contacts c ON c.id = l.contact_id
		 WHERE c.phone = '` + phone + `'`,
		`DELETE l FROM leads l
		 INNER JOIN contacts c ON c.id = l.contact_id
		 WHERE c.phone = '` + phone + `'`,
		// Conversation state (AI) for the conversation.
		`DELETE s FROM conversation_states s
		 INNER JOIN conversations cv ON cv.id = s.conversation_id
		 INNER JOIN contacts c ON c.id = cv.contact_id
		 WHERE c.phone = '` + phone + `'`,
		// Messages + conversations for the contact.
		`DELETE msg FROM messages msg
		 INNER JOIN conversations cv ON cv.id = msg.conversation_id
		 INNER JOIN contacts c ON c.id = cv.contact_id
		 WHERE c.phone = '` + phone + `'`,
		`DELETE cv FROM conversations cv
		 INNER JOIN contacts c ON c.id = cv.contact_id
		 WHERE c.phone = '` + phone + `'`,
		`DELETE FROM contacts WHERE phone = '` + phone + `'`,
	}
	for _, stmt := range statements {
		if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("cleanup: %q: %v", stmt, err)
		}
	}
}
