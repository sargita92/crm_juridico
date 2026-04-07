package automation

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/automation/application"
	"github.com/sasrgita/crm-juridico/internal/automation/application/executors"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	autohttp "github.com/sasrgita/crm-juridico/internal/automation/interfaces/http"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	notifapp "github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ModuleDeps holds all cross-module dependencies for the automation module.
type ModuleDeps struct {
	MoveLeadUC     *funnelapp.MoveLeadUseCase
	LeadRepo       funneldomain.LeadRepository
	ColumnRepo     funneldomain.ColumnRepository
	NoteRepo       funneldomain.LeadNoteRepository
	NotifyService  *notifapp.NotifyService
	DB             *gorm.DB
	ListFunnelsUC  *funnelapp.ListFunnelsUseCase
	SpecialistRepo specialistdomain.SpecialistRepository
	SpecTenantRepo specialistdomain.SpecialistTenantRepository
}

// Module is the automation bounded-context root.
type Module struct {
	handler     *autohttp.Handler
	pageHandler *autohttp.PageHandler
	engine      *application.AutomationEngine
	ticker      *application.ExpirationTicker
}

// NewModule wires all automation dependencies and returns a ready Module.
func NewModule(db *gorm.DB, deps ModuleDeps, log *zap.Logger) *Module {
	// Repositories
	autoRepo := infrastructure.NewGormAutomationRepository(db)
	logRepo := infrastructure.NewGormExecutionLogRepository(db)

	// Adapters
	leadMover := infrastructure.NewLeadMoverAdapter(deps.MoveLeadUC)
	leadFinder := infrastructure.NewLeadFinderAdapter(deps.LeadRepo, db)
	noteSaver := infrastructure.NewNoteSaverAdapter(deps.NoteRepo)
	lostColFinder := infrastructure.NewLostColumnFinderAdapter(deps.ColumnRepo)
	leadDeleter := infrastructure.NewLeadDeleterAdapter(db)
	notifier := infrastructure.NewNotifierAdapter(deps.NotifyService)

	// Stubs for integrations not yet wired
	msgSender := infrastructure.NewMessageSenderStub(log)
	specSwitcher := infrastructure.NewSpecialistSwitcherStub(log)
	productRouter := infrastructure.NewProductRouterStub(log)
	specForProduct := infrastructure.NewSpecialistForProductStub(log)

	// Executors
	moveFunnelExec := executors.NewMoveFunnelExecutor(leadMover)
	autoNoteExec := executors.NewAutoNoteExecutor(noteSaver)
	autoMsgExec := executors.NewAutoMessageExecutor(msgSender, leadFinder)
	switchSpecExec := executors.NewSwitchSpecialistExecutor(specSwitcher, leadFinder)
	detectProdExec := executors.NewDetectProductExecutor(leadFinder, productRouter, leadMover, specSwitcher, specForProduct)
	expirationExec := executors.NewExpirationExecutor(leadFinder, leadMover, leadDeleter, lostColFinder)

	// Suppress "declared but not used" for notifier — it will be used when executor is added
	_ = notifier

	// Engine
	engine := application.NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeMoveFunnel, moveFunnelExec)
	engine.RegisterExecutor(domain.TypeAutoNote, autoNoteExec)
	engine.RegisterExecutor(domain.TypeAutoMessage, autoMsgExec)
	engine.RegisterExecutor(domain.TypeSwitchSpecialist, switchSpecExec)
	engine.RegisterExecutor(domain.TypeDetectProduct, detectProdExec)

	// Expiration ticker
	ticker := application.NewExpirationTicker(autoRepo, logRepo, expirationExec)
	ticker.Start()

	// CRUD + handler
	crudUC := application.NewCRUDUseCase(autoRepo, logRepo)
	handler := autohttp.NewHandler(crudUC, log)
	pageHandler := autohttp.NewPageHandler(crudUC, deps.ListFunnelsUC, deps.ColumnRepo, deps.SpecialistRepo, deps.SpecTenantRepo, log)

	return &Module{handler: handler, pageHandler: pageHandler, engine: engine, ticker: ticker}
}

// Name implements module.Module.
func (m *Module) Name() string { return "automation" }

// RegisterRoutes implements module.Module.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
	m.pageHandler.RegisterPageRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
}

// Engine exposes the AutomationEngine so the funnel module can call OnLeadEvent.
func (m *Module) Engine() *application.AutomationEngine { return m.engine }
