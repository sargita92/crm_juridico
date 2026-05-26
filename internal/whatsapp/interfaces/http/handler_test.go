package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// --- Mocks ---

type mockConvRepo struct {
	conversations map[string]*domain.Conversation
	byContact     map[string]*domain.Conversation
}

func newMockConvRepo() *mockConvRepo {
	return &mockConvRepo{
		conversations: make(map[string]*domain.Conversation),
		byContact:     make(map[string]*domain.Conversation),
	}
}

func (m *mockConvRepo) Create(_ context.Context, conv *domain.Conversation) error {
	m.conversations[conv.ID] = conv
	m.byContact[conv.TenantID+"|"+conv.ContactID] = conv
	return nil
}

func (m *mockConvRepo) FindByID(_ context.Context, id string) (*domain.Conversation, error) {
	if c, ok := m.conversations[id]; ok {
		return c, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConvRepo) FindByContactID(_ context.Context, tenantID, contactID string) (*domain.Conversation, error) {
	if c, ok := m.byContact[tenantID+"|"+contactID]; ok {
		return c, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConvRepo) Update(_ context.Context, conv *domain.Conversation) error {
	m.conversations[conv.ID] = conv
	return nil
}

func (m *mockConvRepo) FindByTenantID(_ context.Context, tenantID string, _ domain.ConversationFilter) (*domain.ConversationList, error) {
	var result []domain.ConversationWithContact
	for _, c := range m.conversations {
		if c.TenantID == tenantID {
			result = append(result, domain.ConversationWithContact{
				Conversation: *c,
				Contact:      domain.Contact{Name: "Test Contact", Phone: "+5511999990001"},
			})
		}
	}
	return &domain.ConversationList{
		Conversations: result,
		Total:         int64(len(result)),
		Page:          1,
		Limit:         20,
	}, nil
}

type mockMsgRepo struct {
	messages map[string]*domain.Message
	byConv   map[string][]*domain.Message
}

func newMockMsgRepo() *mockMsgRepo {
	return &mockMsgRepo{
		messages: make(map[string]*domain.Message),
		byConv:   make(map[string][]*domain.Message),
	}
}

func (m *mockMsgRepo) Create(_ context.Context, msg *domain.Message) error {
	m.messages[msg.ID] = msg
	m.byConv[msg.ConversationID] = append(m.byConv[msg.ConversationID], msg)
	return nil
}

func (m *mockMsgRepo) FindByConversationID(_ context.Context, convID string, _ domain.MessageFilter) ([]domain.Message, error) {
	msgs := m.byConv[convID]
	result := make([]domain.Message, len(msgs))
	for i, msg := range msgs {
		result[i] = *msg
	}
	return result, nil
}

func (m *mockMsgRepo) FindByWhatsAppMsgID(_ context.Context, _ string) (*domain.Message, error) {
	return nil, domain.ErrMessageNotFound
}

func (m *mockMsgRepo) Update(_ context.Context, msg *domain.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

type mockContactRepo struct {
	contacts map[string]*domain.Contact
}

func newMockContactRepo() *mockContactRepo {
	return &mockContactRepo{contacts: make(map[string]*domain.Contact)}
}

func (m *mockContactRepo) Create(_ context.Context, c *domain.Contact) error {
	m.contacts[c.ID] = c
	return nil
}

func (m *mockContactRepo) FindByID(_ context.Context, id string) (*domain.Contact, error) {
	if c, ok := m.contacts[id]; ok {
		return c, nil
	}
	return nil, domain.ErrContactNotFound
}

func (m *mockContactRepo) FindByWhatsAppID(_ context.Context, _, _ string) (*domain.Contact, error) {
	return nil, domain.ErrContactNotFound
}

func (m *mockContactRepo) Update(_ context.Context, c *domain.Contact) error {
	m.contacts[c.ID] = c
	return nil
}

type mockProvider struct {
	connected map[string]bool
	sendErr   error
}

func newMockProvider() *mockProvider {
	return &mockProvider{connected: make(map[string]bool)}
}

func (m *mockProvider) Connect(_ context.Context, tenantID string) (<-chan string, error) {
	m.connected[tenantID] = true
	ch := make(chan string, 1)
	ch <- "qr-test"
	return ch, nil
}

func (m *mockProvider) Disconnect(_ context.Context, tenantID string) error {
	delete(m.connected, tenantID)
	return nil
}

func (m *mockProvider) IsConnected(tenantID string) bool { return m.connected[tenantID] }

func (m *mockProvider) SendMessage(_ context.Context, _, _, _ string) (string, error) {
	if m.sendErr != nil {
		return "", m.sendErr
	}
	return "wa-sent-id", nil
}

func (m *mockProvider) SetMessageHandler(_ domain.IncomingMessageHandler) {}

type mockEventBus struct{ published []events.Event }

func newMockEventBus() *mockEventBus { return &mockEventBus{} }

func (m *mockEventBus) Publish(e events.Event) { m.published = append(m.published, e) }

func (m *mockEventBus) Subscribe(_ string) (<-chan events.Event, func()) {
	ch := make(chan events.Event, 100)
	return ch, func() { close(ch) }
}

// --- Test setup ---

type testDeps struct {
	handler     *Handler
	convRepo    *mockConvRepo
	msgRepo     *mockMsgRepo
	contactRepo *mockContactRepo
	provider    *mockProvider
	eventBus    *mockEventBus
	router      *gin.Engine
}

func setupTest() *testDeps {
	gin.SetMode(gin.TestMode)

	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	contactRepo := newMockContactRepo()
	provider := newMockProvider()
	provider.connected["tenant-1"] = true
	eventBus := newMockEventBus()

	sendUC := application.NewSendMessageUseCase(convRepo, msgRepo, contactRepo, provider, eventBus)
	listUC := application.NewListConversationsUseCase(convRepo)
	getUC := application.NewGetConversationMessagesUseCase(convRepo, msgRepo)
	connectUC := application.NewConnectWhatsAppUseCase(provider)
	statusUC := application.NewGetConnectionStatusUseCase(provider)
	disconnectUC := application.NewDisconnectWhatsAppUseCase(provider)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(sendUC, listUC, getUC, connectUC, statusUC, disconnectUC, testLog)

	router := gin.New()

	// Stub templates
	tmpl := template.New("")
	for _, name := range []string{
		"whatsapp/page.html", "whatsapp/qr.html", "whatsapp/status.html",
		"whatsapp/conversations.html", "whatsapp/chat.html", "whatsapp/message.html",
		"whatsapp/messages_fragment.html", "whatsapp/notes_panel.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	// Fake auth middleware (sets user_id) + fake tenant middleware
	authMw := func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() }
	tenantMw := func(c *gin.Context) { c.Next() }

	handler.RegisterRoutes(router, authMw, tenantMw)

	return &testDeps{
		handler: handler, convRepo: convRepo, msgRepo: msgRepo,
		contactRepo: contactRepo, provider: provider, eventBus: eventBus,
		router: router,
	}
}

func makeRequest(method, path string, body ...string) *http.Request {
	var req *http.Request
	if len(body) > 0 {
		req = httptest.NewRequest(method, path, strings.NewReader(body[0]))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	// Inject tenant ID using the middleware helper
	req = req.WithContext(middleware.SetTenantIDForTest(req.Context(), "tenant-1"))
	return req
}

// --- Tests ---

func TestRenderPage(t *testing.T) {
	deps := setupTest()

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderConversations_Empty(t *testing.T) {
	deps := setupTest()

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderConversations_WithData(t *testing.T) {
	deps := setupTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = deps.convRepo.Create(context.Background(), conv)

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderChat_NotFound(t *testing.T) {
	deps := setupTest()

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/nonexistent"))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenderChat_Success(t *testing.T) {
	deps := setupTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = deps.convRepo.Create(context.Background(), conv)

	msg, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Ola", domain.MessageTypeText, "wa-1", time.Now())
	_ = deps.msgRepo.Create(context.Background(), msg)

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/"+conv.ID))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderChat_WrongTenant(t *testing.T) {
	deps := setupTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "other-tenant", "contact-1")
	_ = deps.convRepo.Create(context.Background(), conv)

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/"+conv.ID))

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSendMessage_Success(t *testing.T) {
	deps := setupTest()

	contact, _ := domain.NewContact(uuid.New().String(), "tenant-1", "Maria", "+55", "jid@wa")
	_ = deps.contactRepo.Create(context.Background(), contact)

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", contact.ID)
	_ = deps.convRepo.Create(context.Background(), conv)

	form := url.Values{"content": {"Ola Maria!"}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/"+conv.ID+"/messages", form.Encode()))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, deps.msgRepo.messages, 1)
}

func TestHandleSendMessage_EmptyContent(t *testing.T) {
	deps := setupTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = deps.convRepo.Create(context.Background(), conv)

	form := url.Values{"content": {""}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/"+conv.ID+"/messages", form.Encode()))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDisconnect(t *testing.T) {
	deps := setupTest()
	require.True(t, deps.provider.IsConnected("tenant-1"))

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/disconnect"))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, deps.provider.IsConnected("tenant-1"))
}

func TestRenderStatus(t *testing.T) {
	deps := setupTest()

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/status"))

	assert.Equal(t, http.StatusOK, w.Code)
}
