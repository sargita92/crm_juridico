package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// --- Mock ContactRepository ---

type mockContactRepo struct {
	contacts   map[string]*domain.Contact
	byWhatsApp map[string]*domain.Contact // key: tenantID+whatsappID
	createErr  error
}

func newMockContactRepo() *mockContactRepo {
	return &mockContactRepo{
		contacts:   make(map[string]*domain.Contact),
		byWhatsApp: make(map[string]*domain.Contact),
	}
}

func (m *mockContactRepo) Create(_ context.Context, contact *domain.Contact) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.contacts[contact.ID] = contact
	m.byWhatsApp[contact.TenantID+"|"+contact.WhatsAppID] = contact
	return nil
}

func (m *mockContactRepo) FindByID(_ context.Context, id string) (*domain.Contact, error) {
	if c, ok := m.contacts[id]; ok {
		return c, nil
	}
	return nil, domain.ErrContactNotFound
}

func (m *mockContactRepo) FindByWhatsAppID(_ context.Context, tenantID, whatsappID string) (*domain.Contact, error) {
	if c, ok := m.byWhatsApp[tenantID+"|"+whatsappID]; ok {
		return c, nil
	}
	return nil, domain.ErrContactNotFound
}

func (m *mockContactRepo) Update(_ context.Context, contact *domain.Contact) error {
	m.contacts[contact.ID] = contact
	m.byWhatsApp[contact.TenantID+"|"+contact.WhatsAppID] = contact
	return nil
}

// --- Mock ConversationRepository ---

type mockConversationRepo struct {
	conversations map[string]*domain.Conversation
	byContact     map[string]*domain.Conversation // key: tenantID+contactID
	createErr     error
}

func newMockConversationRepo() *mockConversationRepo {
	return &mockConversationRepo{
		conversations: make(map[string]*domain.Conversation),
		byContact:     make(map[string]*domain.Conversation),
	}
}

func (m *mockConversationRepo) Create(_ context.Context, conv *domain.Conversation) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.conversations[conv.ID] = conv
	m.byContact[conv.TenantID+"|"+conv.ContactID] = conv
	return nil
}

func (m *mockConversationRepo) FindByID(_ context.Context, id string) (*domain.Conversation, error) {
	if c, ok := m.conversations[id]; ok {
		return c, nil
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConversationRepo) FindByContactID(_ context.Context, tenantID, contactID string) (*domain.Conversation, error) {
	if c, ok := m.byContact[tenantID+"|"+contactID]; ok {
		if c.Status == domain.ConversationStatusOpen {
			return c, nil
		}
	}
	return nil, domain.ErrConversationNotFound
}

func (m *mockConversationRepo) Update(_ context.Context, conv *domain.Conversation) error {
	m.conversations[conv.ID] = conv
	m.byContact[conv.TenantID+"|"+conv.ContactID] = conv
	return nil
}

func (m *mockConversationRepo) FindByTenantID(_ context.Context, tenantID string, filter domain.ConversationFilter) (*domain.ConversationList, error) {
	var result []domain.ConversationWithContact
	for _, c := range m.conversations {
		if c.TenantID == tenantID {
			result = append(result, domain.ConversationWithContact{
				Conversation: *c,
			})
		}
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}
	return &domain.ConversationList{
		Conversations: result,
		Total:         int64(len(result)),
		Page:          page,
		Limit:         limit,
	}, nil
}

// --- Mock MessageRepository ---

type mockMessageRepo struct {
	messages  map[string]*domain.Message
	byConvID  map[string][]*domain.Message
	byWaMsgID map[string]*domain.Message
	createErr error
}

func newMockMessageRepo() *mockMessageRepo {
	return &mockMessageRepo{
		messages:  make(map[string]*domain.Message),
		byConvID:  make(map[string][]*domain.Message),
		byWaMsgID: make(map[string]*domain.Message),
	}
}

func (m *mockMessageRepo) Create(_ context.Context, msg *domain.Message) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.messages[msg.ID] = msg
	m.byConvID[msg.ConversationID] = append(m.byConvID[msg.ConversationID], msg)
	if msg.WhatsAppMsgID != "" {
		m.byWaMsgID[msg.WhatsAppMsgID] = msg
	}
	return nil
}

func (m *mockMessageRepo) FindByConversationID(_ context.Context, conversationID string, filter domain.MessageFilter) ([]domain.Message, error) {
	msgs := m.byConvID[conversationID]
	var result []domain.Message
	for _, msg := range msgs {
		result = append(result, *msg)
	}
	return result, nil
}

func (m *mockMessageRepo) FindByWhatsAppMsgID(_ context.Context, whatsappMsgID string) (*domain.Message, error) {
	if msg, ok := m.byWaMsgID[whatsappMsgID]; ok {
		return msg, nil
	}
	return nil, domain.ErrMessageNotFound
}

func (m *mockMessageRepo) Update(_ context.Context, msg *domain.Message) error {
	m.messages[msg.ID] = msg
	return nil
}

func (m *mockMessageRepo) DeleteByConversationID(_ context.Context, conversationID string) (int64, error) {
	msgs := m.byConvID[conversationID]
	for _, msg := range msgs {
		delete(m.messages, msg.ID)
		if msg.WhatsAppMsgID != "" {
			delete(m.byWaMsgID, msg.WhatsAppMsgID)
		}
	}
	delete(m.byConvID, conversationID)
	return int64(len(msgs)), nil
}

// --- Mock WhatsAppProvider ---

type mockProvider struct {
	connected   map[string]bool
	sendErr     error
	connectErr  error
	lastSentTo  string
	lastSentMsg string
	handler     domain.IncomingMessageHandler
	mu          sync.Mutex
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		connected: make(map[string]bool),
	}
}

func (m *mockProvider) Connect(_ context.Context, tenantID string) (<-chan string, error) {
	if m.connectErr != nil {
		return nil, m.connectErr
	}
	m.mu.Lock()
	m.connected[tenantID] = true
	m.mu.Unlock()
	ch := make(chan string, 1)
	ch <- "qr-code-data"
	return ch, nil
}

func (m *mockProvider) Disconnect(_ context.Context, tenantID string) error {
	m.mu.Lock()
	delete(m.connected, tenantID)
	m.mu.Unlock()
	return nil
}

func (m *mockProvider) IsConnected(tenantID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected[tenantID]
}

func (m *mockProvider) SendMessage(_ context.Context, tenantID, recipientWhatsAppID, content string) (string, error) {
	if m.sendErr != nil {
		return "", m.sendErr
	}
	m.mu.Lock()
	m.lastSentTo = recipientWhatsAppID
	m.lastSentMsg = content
	m.mu.Unlock()
	return "wa-sent-" + time.Now().Format("150405"), nil
}

func (m *mockProvider) SetMessageHandler(handler domain.IncomingMessageHandler) {
	m.handler = handler
}

// --- Mock EventBus ---

type mockEventBus struct {
	mu     sync.Mutex
	events []events.Event
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{}
}

func (m *mockEventBus) Publish(event events.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventBus) Subscribe(tenantID string) (<-chan events.Event, func()) {
	ch := make(chan events.Event, 100)
	return ch, func() { close(ch) }
}

var errProviderSend = errors.New("provider send failed")
