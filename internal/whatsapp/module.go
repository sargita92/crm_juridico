package whatsapp

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/infrastructure"
	whatsapphttp "github.com/sasrgita/crm-juridico/internal/whatsapp/interfaces/http"
)

type Module struct {
	handler          *whatsapphttp.Handler
	provider         domain.WhatsAppProvider
	receiveMessageUC *application.ReceiveMessageUseCase
	sendMessageUC    *application.SendMessageUseCase
	contactRepo      domain.ContactRepository
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
}

func NewModule(db *gorm.DB, provider domain.WhatsAppProvider, eventBus events.EventBus, log *zap.Logger) *Module {
	contactRepo := infrastructure.NewGormContactRepository(db)
	conversationRepo := infrastructure.NewGormConversationRepository(db)
	messageRepo := infrastructure.NewGormMessageRepository(db)

	receiveMessageUC := application.NewReceiveMessageUseCase(contactRepo, conversationRepo, messageRepo, eventBus, log)
	sendMessageUC := application.NewSendMessageUseCase(conversationRepo, messageRepo, contactRepo, provider, eventBus)
	listConversationsUC := application.NewListConversationsUseCase(conversationRepo)
	getMessagesUC := application.NewGetConversationMessagesUseCase(conversationRepo, messageRepo)
	connectUC := application.NewConnectWhatsAppUseCase(provider)
	statusUC := application.NewGetConnectionStatusUseCase(provider)
	disconnectUC := application.NewDisconnectWhatsAppUseCase(provider)

	provider.SetMessageHandler(func(ctx context.Context, event domain.IncomingMessage) {
		_ = receiveMessageUC.Execute(ctx, event)
	})

	handler := whatsapphttp.NewHandler(
		sendMessageUC, listConversationsUC, getMessagesUC,
		connectUC, statusUC, disconnectUC, eventBus, log,
	)

	return &Module{
		handler:          handler,
		provider:         provider,
		receiveMessageUC: receiveMessageUC,
		sendMessageUC:    sendMessageUC,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
	}
}

func (m *Module) Name() string { return "whatsapp" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
}

func (m *Module) SetLeadCreator(lc domain.LeadCreator) {
	m.receiveMessageUC.SetLeadCreator(lc)
}

// SetLeadNotesService wires the funnel-backed notes service into the HTTP handler so
// the chat can show and add notes for the lead a conversation is currently on.
func (m *Module) SetLeadNotesService(s domain.LeadNotesService) {
	m.handler.SetNotesService(s)
}

func (m *Module) SetFileStorer(fs domain.FileStorer) {
	m.receiveMessageUC.SetFileStorer(fs)
}

func (m *Module) ContactRepo() domain.ContactRepository {
	return m.contactRepo
}

func (m *Module) ConversationRepo() domain.ConversationRepository {
	return m.conversationRepo
}

func (m *Module) MessageRepo() domain.MessageRepository {
	return m.messageRepo
}

func (m *Module) SendMessageUC() *application.SendMessageUseCase {
	return m.sendMessageUC
}

func (m *Module) ReceiveMessageUC() *application.ReceiveMessageUseCase {
	return m.receiveMessageUC
}

func (m *Module) SetAIHandler(handler domain.AIHandler) {
	m.receiveMessageUC.SetAIHandler(handler)
}
