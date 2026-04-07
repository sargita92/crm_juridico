package ai

import (
	"context"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/ai/infrastructure"
	docDomain "github.com/sasrgita/crm-juridico/internal/document/domain"
	funnelApp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productApp "github.com/sasrgita/crm-juridico/internal/product/application"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	whatsappApp "github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// ModuleDeps holds cross-module dependencies needed by the AI module.
type ModuleDeps struct {
	SpecialistRepo  specDomain.SpecialistRepository
	StepRepo        specDomain.StepRepository
	GuardrailRepo   specDomain.GuardrailRepository
	SpecTenantRepo  specDomain.SpecialistTenantRepository
	DocRepo         docDomain.DocumentRepository
	SpecDocRepo     docDomain.SpecialistDocumentRepository
	ProductRepo     productDomain.ProductRepository
	PhoneNumberRepo productDomain.PhoneNumberRepository
	DetectProductUC *productApp.DetectProductUseCase
	MessageRepo     whatsappDomain.MessageRepository
	SendMessageUC   *whatsappApp.SendMessageUseCase
	LeadRepo        funnelDomain.LeadRepository
	MoveLeadUC      *funnelApp.MoveLeadUseCase
}

// conversationContext holds routing context stored between routing and debounce callback.
type conversationContext struct {
	TenantID     string
	SpecialistID string
	ProductID    string
}

// Module wires the AI subsystem and implements module.Module.
type Module struct {
	engine            *application.ConversationEngine
	debouncer         *application.Debouncer
	router            *application.SpecialistRouter
	activateHandoff   *application.ActivateHandoffUseCase
	deactivateHandoff *application.DeactivateHandoffUseCase
	aiConfigRepo      domain.AIConfigRepository
	spProductRepo     domain.SpecialistProductRepository
	stateRepo         domain.ConversationStateRepository
	log               *zap.Logger

	contexts   map[string]conversationContext
	contextsMu sync.RWMutex
}

// NewModule creates and wires all AI components.
func NewModule(db *gorm.DB, cfg config.AIConfigEnv, log *zap.Logger, deps ModuleDeps) *Module {
	// 1. Create AI-specific repos.
	aiConfigRepo := infrastructure.NewGormAIConfigRepository(db)
	convStateRepo := infrastructure.NewGormConversationStateRepository(db)
	spProductRepo := infrastructure.NewGormSpecialistProductRepository(db)

	// 2. Create adapters wrapping cross-module dependencies.
	specialistFinderAdapter := infrastructure.NewSpecialistFinderAdapter(deps.SpecialistRepo)
	stepFinderAdapter := infrastructure.NewStepFinderAdapter(deps.StepRepo)
	guardrailFinderAdapter := infrastructure.NewGuardrailFinderAdapter(deps.GuardrailRepo)
	documentFetcherAdapter := infrastructure.NewDocumentFetcherAdapter(deps.DocRepo, deps.SpecDocRepo)
	productInfoAdapter := infrastructure.NewProductInfoFinderAdapter(deps.ProductRepo)
	messageHistoryAdapter := infrastructure.NewMessageHistoryAdapter(deps.MessageRepo)
	messageSenderAdapter := infrastructure.NewMessageSenderAdapter(deps.SendMessageUC)
	leadUpdaterAdapter := infrastructure.NewLeadUpdaterAdapter(deps.LeadRepo, deps.MoveLeadUC)
	phoneFinderAdapter := infrastructure.NewPhoneNumberFinderAdapter(deps.PhoneNumberRepo)
	productDetectorAdapter := infrastructure.NewProductDetectorAdapter(deps.DetectProductUC)
	defaultSpFinderAdapter := infrastructure.NewDefaultSpecialistFinderAdapter(deps.SpecTenantRepo)

	// 3. Create ProviderRegistry and register providers.
	providerRegistry := domain.NewProviderRegistry()
	openaiProvider := infrastructure.NewOpenAIProvider(cfg.OpenAIAPIKey, "", log)
	providerRegistry.Register(openaiProvider)

	// 4. Create ConfigResolver.
	configResolver := application.NewEnvConfigResolver(aiConfigRepo, cfg)

	// 5. Create ContextBuilder.
	contextBuilder := application.NewContextBuilder(
		specialistFinderAdapter,
		stepFinderAdapter,
		guardrailFinderAdapter,
		documentFetcherAdapter,
		productInfoAdapter,
		messageHistoryAdapter,
	)

	// 6. Create StepEvaluator and GuardrailChecker.
	stepEvaluator := application.NewStepEvaluator()
	guardrailChecker := application.NewGuardrailChecker()

	// 7. Create ConversationEngine.
	engine := application.NewConversationEngine(
		providerRegistry,
		configResolver,
		convStateRepo,
		contextBuilder,
		stepEvaluator,
		guardrailChecker,
		messageSenderAdapter,
		leadUpdaterAdapter,
		log,
	)

	// 8. Create SpecialistRouter.
	specialistRouter := application.NewSpecialistRouter(
		phoneFinderAdapter,
		spProductRepo,
		productDetectorAdapter,
		defaultSpFinderAdapter,
	)

	// 9. Create handoff use cases.
	activateHandoff := application.NewActivateHandoffUseCase(convStateRepo, log)
	deactivateHandoff := application.NewDeactivateHandoffUseCase(convStateRepo, log)

	m := &Module{
		engine:            engine,
		router:            specialistRouter,
		activateHandoff:   activateHandoff,
		deactivateHandoff: deactivateHandoff,
		aiConfigRepo:      aiConfigRepo,
		spProductRepo:     spProductRepo,
		stateRepo:         convStateRepo,
		log:               log,
		contexts:          make(map[string]conversationContext),
	}

	// 10. Create Debouncer with callback that retrieves stored context and calls the engine.
	debouncer := application.NewDebouncer(
		time.Duration(cfg.DefaultDebounce)*time.Second,
		func(conversationID string, messages []string) {
			convCtx, ok := m.getContext(conversationID)
			if !ok {
				log.Warn("ai_module: debounce callback missing context",
					zap.String("conversation_id", conversationID),
				)
				return
			}
			if err := m.engine.HandleMessages(
				context.Background(),
				convCtx.TenantID, conversationID,
				convCtx.SpecialistID, convCtx.ProductID,
				messages,
			); err != nil {
				log.Error("ai_module: engine.HandleMessages failed",
					zap.String("conversation_id", conversationID),
					zap.Error(err),
				)
			}
		},
	)
	m.debouncer = debouncer

	return m
}

// storeContext saves the routing context for a conversation (thread-safe).
func (m *Module) storeContext(conversationID string, ctx conversationContext) {
	m.contextsMu.Lock()
	m.contexts[conversationID] = ctx
	m.contextsMu.Unlock()
}

// getContext retrieves and removes the routing context for a conversation (thread-safe).
func (m *Module) getContext(conversationID string) (conversationContext, bool) {
	m.contextsMu.Lock()
	defer m.contextsMu.Unlock()
	ctx, ok := m.contexts[conversationID]
	if ok {
		delete(m.contexts, conversationID)
	}
	return ctx, ok
}

// HandleIncomingMessage implements whatsappDomain.AIHandler.
// It routes the message to a specialist, stores context, and debounces.
func (m *Module) HandleIncomingMessage(ctx context.Context, tenantID, conversationID, senderPhone, content string) {
	specialistID, productID, err := m.router.Route(ctx, tenantID, senderPhone, content)
	if err != nil {
		m.log.Warn("ai_module: no specialist available",
			zap.String("tenant_id", tenantID),
			zap.String("conversation_id", conversationID),
			zap.Error(err),
		)
		return
	}

	m.storeContext(conversationID, conversationContext{
		TenantID:     tenantID,
		SpecialistID: specialistID,
		ProductID:    productID,
	})

	m.debouncer.Add(conversationID, content)
}

// --- module.Module interface ---

func (m *Module) Name() string { return "ai" }

func (m *Module) RegisterRoutes(_ *gin.Engine, _ module.Middlewares) {
	// HTTP handler will be added in Task 20.
}

// --- accessor methods for main.go wiring ---

func (m *Module) AIConfigRepo() domain.AIConfigRepository {
	return m.aiConfigRepo
}

func (m *Module) SPProductRepo() domain.SpecialistProductRepository {
	return m.spProductRepo
}

func (m *Module) StateRepo() domain.ConversationStateRepository {
	return m.stateRepo
}

func (m *Module) ActivateHandoffUC() *application.ActivateHandoffUseCase {
	return m.activateHandoff
}

func (m *Module) DeactivateHandoffUC() *application.DeactivateHandoffUseCase {
	return m.deactivateHandoff
}
