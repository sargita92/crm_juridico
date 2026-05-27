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
	"github.com/sasrgita/crm-juridico/internal/ai/infrastructure/tools"
	aihttp "github.com/sasrgita/crm-juridico/internal/ai/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/ai/interfaces/http/playground"
	automationApp "github.com/sasrgita/crm-juridico/internal/automation/application"
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

// activateHandoffAdapter adapts ActivateHandoffUseCase to the HandoffActivator interface
// expected by ConversationEngine, bridging the tenantID gap (tenantID is unknown at
// construction time; the adapter passes an empty string for the metric label).
type activateHandoffAdapter struct {
	uc *application.ActivateHandoffUseCase
}

func (a activateHandoffAdapter) Activate(ctx context.Context, conversationID string) error {
	return a.uc.Execute(ctx, "", conversationID)
}

// ModuleDeps holds cross-module dependencies needed by the AI module.
type ModuleDeps struct {
	SpecialistRepo   specDomain.SpecialistRepository
	StepRepo         specDomain.StepRepository
	GuardrailRepo    specDomain.GuardrailRepository
	SpecTenantRepo   specDomain.SpecialistTenantRepository
	DocRepo          docDomain.DocumentRepository
	SpecDocRepo      docDomain.SpecialistDocumentRepository
	ProductRepo      productDomain.ProductRepository
	PhoneNumberRepo  productDomain.PhoneNumberRepository
	DetectProductUC  *productApp.DetectProductUseCase
	MessageRepo      whatsappDomain.MessageRepository
	ConversationRepo whatsappDomain.ConversationRepository
	SendMessageUC    *whatsappApp.SendMessageUseCase
	ReceiveMessageUC *whatsappApp.ReceiveMessageUseCase
	LeadRepo         funnelDomain.LeadRepository
	MoveLeadUC       *funnelApp.MoveLeadUseCase
	FunnelRepo       funnelDomain.FunnelRepository
	ColumnRepo       funnelDomain.ColumnRepository
	// Tool wiring deps (Task 16)
	NoteRepo             funnelDomain.LeadNoteRepository
	TenantProductRepo    productDomain.TenantProductRepository
	AutomationEngine     *automationApp.AutomationEngine
	SpecialistToolFinder application.SpecialistToolFinder
	// ScoringConfigFinder is optional; when provided the engine moves leads to
	// qualified/disqualified columns based on the specialist's scoring threshold.
	ScoringConfigFinder application.ScoringConfigFinder
	// ToolRegistry is an optional pre-created registry shared with the specialist UI (Task 17).
	// When non-nil, tools are registered into it; when nil, a new private registry is created.
	ToolRegistry *application.ToolRegistry
	// CrossSellRuleRepo is optional. When non-nil, the engine auto-builds the
	// CrossSellExecutor using existing deps (ProductRepo, LeadRepo, ColumnRepo, etc.)
	// and evaluates cross-sell rules before invoking the LLM.
	CrossSellRuleRepo specDomain.CrossSellRuleRepository
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
	resetUC           *application.ResetConversationUseCase
	handler           *aihttp.Handler
	playgroundHandler *playground.Handler
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

	// Reset conversation dependencies (F17 Task 9).
	entryFinderAdapter := infrastructure.NewFunnelEntryAdapter(deps.FunnelRepo, deps.ColumnRepo)
	leadResetterAdapter := infrastructure.NewLeadResetterAdapter(deps.LeadRepo)
	resetUC := application.NewResetConversationUseCase(
		convStateRepo,
		entryFinderAdapter,
		leadResetterAdapter,
		messageSenderAdapter,
		log,
	)

	// 3. Create ProviderRegistry and register providers.
	providerRegistry := domain.NewProviderRegistry()
	openaiProvider := infrastructure.NewOpenAIProvider(cfg.OpenAIAPIKey, "", log)
	providerRegistry.Register(openaiProvider)
	providerRegistry.Register(infrastructure.NewFakeProvider())

	// 4. Create ConfigResolver.
	configResolver := application.NewEnvConfigResolver(aiConfigRepo, cfg)

	// 5. Create tool adapters (Task 16).
	leadSearchAdapter := infrastructure.NewLeadSearchAdapter(deps.LeadRepo)
	leadDetailAdapter := infrastructure.NewLeadDetailAdapter(deps.LeadRepo, deps.NoteRepo)
	convHistoryToolAdapter := infrastructure.NewConversationHistoryToolAdapter(deps.MessageRepo)
	productListToolAdapter := infrastructure.NewProductListToolAdapter(deps.TenantProductRepo, deps.ProductRepo)
	pipelineToolAdapter := infrastructure.NewPipelineToolAdapter(deps.FunnelRepo, deps.ColumnRepo, deps.LeadRepo)
	leadMoveToolAdapter := infrastructure.NewLeadMoveToolAdapter(deps.LeadRepo, deps.MoveLeadUC)
	noteCreatorToolAdapter := infrastructure.NewNoteCreatorToolAdapter(deps.LeadRepo, deps.NoteRepo)
	scoreUpdaterToolAdapter := infrastructure.NewScoreUpdaterToolAdapter(deps.LeadRepo)
	automationTriggerToolAdapter := infrastructure.NewAutomationTriggerToolAdapter(deps.AutomationEngine)
	specialistSwitcherToolAdapter := infrastructure.NewSpecialistSwitcherToolAdapter(convStateRepo)

	// 5b. Register all 10 tools into a shared or private registry.
	// When deps.ToolRegistry is provided (from main.go), tools are registered there so the
	// specialist admin UI can list them without a circular dependency.
	toolRegistry := deps.ToolRegistry
	if toolRegistry == nil {
		toolRegistry = application.NewToolRegistry()
	}
	toolRegistry.Register(tools.NewSearchLeadsTool(leadSearchAdapter))
	toolRegistry.Register(tools.NewGetLeadDetailTool(leadDetailAdapter))
	toolRegistry.Register(tools.NewGetConversationHistoryTool(convHistoryToolAdapter))
	toolRegistry.Register(tools.NewListProductsTool(productListToolAdapter))
	toolRegistry.Register(tools.NewGetPipelineTool(pipelineToolAdapter))
	toolRegistry.Register(tools.NewMoveLeadTool(leadMoveToolAdapter))
	toolRegistry.Register(tools.NewCreateLeadNoteTool(noteCreatorToolAdapter))
	toolRegistry.Register(tools.NewUpdateLeadScoreTool(scoreUpdaterToolAdapter))
	toolRegistry.Register(tools.NewTriggerAutomationTool(automationTriggerToolAdapter))
	toolRegistry.Register(tools.NewSwitchSpecialistTool(specialistSwitcherToolAdapter))

	// 5c. Create ToolResolver.
	var toolResolver *application.ToolResolver
	if deps.SpecialistToolFinder != nil {
		toolResolver = application.NewToolResolver(toolRegistry, deps.SpecialistToolFinder)
	}

	// 6. Create ContextBuilder.
	contextBuilder := application.NewContextBuilder(
		specialistFinderAdapter,
		stepFinderAdapter,
		guardrailFinderAdapter,
		documentFetcherAdapter,
		productInfoAdapter,
		messageHistoryAdapter,
		toolResolver,
	)

	// 7. Create StepEvaluator and GuardrailChecker.
	stepEvaluator := application.NewStepEvaluator()
	guardrailChecker := application.NewGuardrailChecker()

	// 9. Create handoff use cases (before engine so activateHandoff can be injected).
	activateHandoff := application.NewActivateHandoffUseCase(convStateRepo, log)
	deactivateHandoff := application.NewDeactivateHandoffUseCase(convStateRepo, log)

	// 8. Create cross-sell components (optional; nil when CrossSellRuleRepo not wired).
	// When CrossSellRuleRepo is provided, the executor is auto-built from existing module deps.
	var crossSellEvaluator *application.CrossSellRuleEvaluator
	var crossSellExecutor *application.CrossSellExecutor
	if deps.CrossSellRuleRepo != nil {
		crossSellEvaluator = application.NewCrossSellRuleEvaluator()

		productNameLookup := infrastructure.NewProductNameLookupAdapter(deps.ProductRepo)
		conversationMover := infrastructure.NewConversationMoverAdapter(convStateRepo)
		leadFactory := infrastructure.NewLeadFactoryAdapter(deps.LeadRepo)

		// ProductSpecialistResolver needs a funnelProductFinder. We use the GORM DB
		// directly via a thin shim embedded in cross_sell_adapters.go.
		productSpecialistResolver := infrastructure.NewProductSpecialistResolverAdapter(
			spProductRepo,
			infrastructure.NewGormFunnelProductFinder(db),
			deps.ColumnRepo,
		)

		crossSellExecutor = application.NewCrossSellExecutor(
			productSpecialistResolver,
			leadFactory,
			conversationMover,
			leadUpdaterAdapter,
			messageSenderAdapter,
			productNameLookup,
		)
	}

	// 8. Create ConversationEngine.
	engine := application.NewConversationEngine(
		providerRegistry,
		configResolver,
		convStateRepo,
		contextBuilder,
		stepEvaluator,
		guardrailChecker,
		messageSenderAdapter,
		leadUpdaterAdapter,
		resetUC,
		cfg.ResetCommandEnabled,
		toolRegistry,
		cfg.ToolResultMaxLength,
		cfg.ToolLoopMaxIterations,
		deps.ScoringConfigFinder,
		activateHandoffAdapter{uc: activateHandoff},
		deps.CrossSellRuleRepo,
		crossSellEvaluator,
		crossSellExecutor,
		log,
	)

	// 8. Create SpecialistRouter.
	specialistRouter := application.NewSpecialistRouter(
		phoneFinderAdapter,
		spProductRepo,
		productDetectorAdapter,
		defaultSpFinderAdapter,
	)

	// 10. Create ProductListerAdapter and HTTP handler.
	productListerAdapter := infrastructure.NewProductListerAdapter(deps.ProductRepo)
	handler := aihttp.NewHandler(
		aiConfigRepo,
		spProductRepo,
		convStateRepo,
		activateHandoff,
		deactivateHandoff,
		productListerAdapter,
		log,
	)

	// 10b. Conditionally wire the AI playground dev routes.
	var playgroundHandler *playground.Handler
	if cfg.PlaygroundEnabled {
		contactAdapter := infrastructure.NewPlaygroundContactAdapter(deps.ConversationRepo)
		messageAdapter := infrastructure.NewPlaygroundMessageAdapter(deps.MessageRepo)
		playgroundHandler = playground.NewHandler(
			contactAdapter,
			messageAdapter,
			deps.ReceiveMessageUC,
			resetUC,
			messageAdapter,
			log,
		)
		log.Info("ai playground: ENABLED — dev routes registered at /tenant/ai/playground")
	}

	m := &Module{
		engine:            engine,
		router:            specialistRouter,
		activateHandoff:   activateHandoff,
		deactivateHandoff: deactivateHandoff,
		aiConfigRepo:      aiConfigRepo,
		spProductRepo:     spProductRepo,
		stateRepo:         convStateRepo,
		resetUC:           resetUC,
		handler:           handler,
		playgroundHandler: playgroundHandler,
		log:               log,
		contexts:          make(map[string]conversationContext),
	}

	// 11. Create Debouncer with callback that retrieves stored context and calls the engine.
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

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw)
	if m.playgroundHandler != nil {
		m.playgroundHandler.RegisterRoutes(router, mw)
	}
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

func (m *Module) ResetConversationUC() *application.ResetConversationUseCase {
	return m.resetUC
}
