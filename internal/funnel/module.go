package funnel

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/funnel/infrastructure"
	funnelhttp "github.com/sasrgita/crm-juridico/internal/funnel/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type Module struct {
	handler          *funnelhttp.Handler
	leadCreator      *application.CreateLeadUseCase
	listFunnelsUC    *application.ListFunnelsUseCase
	moveLeadUC       *application.MoveLeadUseCase
	assignLeadUC     *application.AssignLeadUseCase
	leadRepo         domain.LeadRepository
	noteRepo         domain.LeadNoteRepository
	columnRepo       domain.ColumnRepository
	funnelRepo       domain.FunnelRepository
	createLeadNoteUC *application.CreateLeadNoteUseCase
	userNameProvider domain.UserNameProvider
}

func NewModule(
	db *gorm.DB,
	contactProvider domain.ContactProvider,
	messageProvider domain.MessageProvider,
	userNameProvider domain.UserNameProvider,
	productDetector domain.ProductDetector,
	productProvider domain.ProductProvider,
	funnelProductRouter domain.FunnelProductRouter,
	productLister domain.ProductLister,
	eventBus events.EventBus,
	picker domain.ResponsiblePicker,
	log *zap.Logger,
) *Module {
	funnelRepo := infrastructure.NewGormFunnelRepository(db)
	columnRepo := infrastructure.NewGormColumnRepository(db)
	leadRepo := infrastructure.NewGormLeadRepository(db)
	movementRepo := infrastructure.NewGormLeadMovementRepository(db)
	noteRepo := infrastructure.NewGormLeadNoteRepository(db)

	// Use cases
	getKanbanUC := application.NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)
	listFunnelsUC := application.NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)
	getFunnelUC := application.NewGetFunnelUseCase(funnelRepo, columnRepo)
	createFunnelUC := application.NewCreateFunnelUseCase(funnelRepo, columnRepo)
	updateFunnelUC := application.NewUpdateFunnelUseCase(funnelRepo)
	toggleFunnelUC := application.NewToggleFunnelUseCase(funnelRepo)
	createColumnUC := application.NewCreateColumnUseCase(funnelRepo, columnRepo)
	deleteColumnUC := application.NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)
	moveColumnUC := application.NewMoveColumnUseCase(funnelRepo, columnRepo)
	createLeadUC := application.NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, productDetector, funnelProductRouter, eventBus, picker)
	moveLeadUC := application.NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, eventBus)
	moveLeadUC.SetAuditLogger(log)
	getLeadDetailUC := application.NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo, userNameProvider, productProvider)
	getLeadDetailUC.SetAuditLogger(log)
	createLeadNoteUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)
	assignLeadUC := application.NewAssignLeadUseCase(leadRepo)

	handler := funnelhttp.NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC, assignLeadUC,
		leadRepo, productLister, productProvider, log,
	)

	return &Module{
		handler:          handler,
		leadCreator:      createLeadUC,
		listFunnelsUC:    listFunnelsUC,
		moveLeadUC:       moveLeadUC,
		assignLeadUC:     assignLeadUC,
		leadRepo:         leadRepo,
		noteRepo:         noteRepo,
		columnRepo:       columnRepo,
		funnelRepo:       funnelRepo,
		createLeadNoteUC: createLeadNoteUC,
		userNameProvider: userNameProvider,
	}
}

func (m *Module) Name() string { return "funnel" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
}

func (m *Module) LeadCreator() *application.CreateLeadUseCase {
	return m.leadCreator
}

func (m *Module) ListFunnelsUC() *application.ListFunnelsUseCase {
	return m.listFunnelsUC
}

func (m *Module) LeadRepo() domain.LeadRepository {
	return m.leadRepo
}

// LeadLoadCounter exposes the gorm lead repository as a LeadLoadCounter so
// that cross-module adapters (e.g. auth's LoadBalancePicker) can evaluate the
// "least load" algorithm without depending on the concrete gorm type.
func (m *Module) LeadLoadCounter() domain.LeadLoadCounter {
	return m.leadRepo.(domain.LeadLoadCounter)
}

func (m *Module) MoveLeadUC() *application.MoveLeadUseCase {
	return m.moveLeadUC
}

func (m *Module) AssignLeadUC() *application.AssignLeadUseCase {
	return m.assignLeadUC
}

func (m *Module) NoteRepo() domain.LeadNoteRepository {
	return m.noteRepo
}

// LeadNotesService returns an adapter implementing the whatsapp module's
// LeadNotesService port, giving the WhatsApp chat access to the notes of the lead a
// conversation is currently on.
func (m *Module) LeadNotesService(conversations infrastructure.ConversationFinder) whatsappdomain.LeadNotesService {
	return infrastructure.NewWhatsAppNotesAdapter(
		m.leadRepo, m.noteRepo, m.userNameProvider, m.createLeadNoteUC,
		conversations, m.leadCreator,
	)
}

func (m *Module) ColumnRepo() domain.ColumnRepository {
	return m.columnRepo
}

func (m *Module) FunnelRepo() domain.FunnelRepository {
	return m.funnelRepo
}

// LeadLookupAdapter returns an adapter implementing the files module's
// LeadLookup port. Used to associate captured WhatsApp media with the lead
// owning the originating conversation.
func (m *Module) LeadLookupAdapter() *infrastructure.FileLeadLookupAdapter {
	return infrastructure.NewFileLeadLookupAdapter(m.leadRepo)
}

// SetAutomationTrigger wires an AutomationTrigger into both CreateLead and MoveLead
// use cases so that automation rules are fired on lead events.
func (m *Module) SetAutomationTrigger(t domain.AutomationTrigger) {
	m.leadCreator.SetAutomationTrigger(t)
	m.moveLeadUC.SetAutomationTrigger(t)
}

// SetResponsiblePicker late-binds the ResponsiblePicker into the CreateLead
// use case. Required because the picker depends on repositories owned by the
// permission and auth modules, which in turn depend on funnel — breaking the
// construction cycle via this setter.
func (m *Module) SetResponsiblePicker(p domain.ResponsiblePicker) {
	m.leadCreator.SetPicker(p)
}
