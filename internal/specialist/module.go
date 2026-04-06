package specialist

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
	specialisthttp "github.com/sasrgita/crm-juridico/internal/specialist/interfaces/http"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type Module struct {
	specialistRepo   domain.SpecialistRepository
	handler          *specialisthttp.Handler
	guardrailHandler *specialisthttp.GuardrailHandler
	stepHandler      *specialisthttp.StepHandler
	scoringHandler   *specialisthttp.ScoringHandler
}

func NewModule(db *gorm.DB, tenantRepo tenantdomain.TenantRepository) *Module {
	specialistRepo := infrastructure.NewGormSpecialistRepository(db)
	specialistTenantRepo := infrastructure.NewGormSpecialistTenantRepository(db)

	// Specialist CRUD use cases
	createUC := application.NewCreateSpecialistUseCase(specialistRepo)
	listUC := application.NewListSpecialistsUseCase(specialistRepo, specialistTenantRepo)
	getUC := application.NewGetSpecialistUseCase(specialistRepo)
	updateUC := application.NewUpdateSpecialistUseCase(specialistRepo)
	deactivateUC := application.NewDeactivateSpecialistUseCase(specialistRepo)
	activateUC := application.NewActivateSpecialistUseCase(specialistRepo)
	associateUC := application.NewAssociateTenantUseCase(specialistRepo, tenantRepo, specialistTenantRepo)
	dissociateUC := application.NewDissociateTenantUseCase(specialistRepo, specialistTenantRepo)
	listTenantsUC := application.NewListSpecialistTenantsUseCase(specialistRepo, specialistTenantRepo, tenantRepo)
	listAvailableUC := application.NewListAvailableTenantsUseCase(specialistTenantRepo, tenantRepo)

	handler := specialisthttp.NewHandler(
		createUC, listUC, getUC,
		updateUC, deactivateUC, activateUC,
		associateUC, dissociateUC,
		listTenantsUC, listAvailableUC,
	)

	// Guardrail use cases
	guardrailRepo := infrastructure.NewGormGuardrailRepository(db)
	createGuardrailUC := application.NewCreateGuardrailUseCase(specialistRepo, guardrailRepo)
	updateGuardrailUC := application.NewUpdateGuardrailUseCase(guardrailRepo)
	toggleGuardrailUC := application.NewToggleGuardrailUseCase(guardrailRepo)
	deleteGuardrailUC := application.NewDeleteGuardrailUseCase(guardrailRepo)
	listGuardrailsUC := application.NewListGuardrailsUseCase(guardrailRepo)

	guardrailHandler := specialisthttp.NewGuardrailHandler(
		createGuardrailUC, updateGuardrailUC, toggleGuardrailUC,
		deleteGuardrailUC, listGuardrailsUC,
	)

	// Step use cases
	stepRepo := infrastructure.NewGormStepRepository(db)
	createStepUC := application.NewCreateStepUseCase(specialistRepo, stepRepo)
	updateStepUC := application.NewUpdateStepUseCase(stepRepo)
	deleteStepUC := application.NewDeleteStepUseCase(stepRepo)
	moveStepUC := application.NewMoveStepUseCase(stepRepo)
	listStepsUC := application.NewListStepsUseCase(stepRepo)

	stepHandler := specialisthttp.NewStepHandler(
		createStepUC, updateStepUC, deleteStepUC, moveStepUC, listStepsUC,
	)

	// Scoring use cases
	scoringRepo := infrastructure.NewGormScoringConfigRepository(db)
	getScoringUC := application.NewGetScoringUseCase(stepRepo, scoringRepo)
	updateScoringUC := application.NewUpdateScoringUseCase(stepRepo, scoringRepo)

	scoringHandler := specialisthttp.NewScoringHandler(getScoringUC, updateScoringUC)

	return &Module{
		specialistRepo:   specialistRepo,
		handler:          handler,
		guardrailHandler: guardrailHandler,
		stepHandler:      stepHandler,
		scoringHandler:   scoringHandler,
	}
}

func (m *Module) Name() string { return "specialist" }

func (m *Module) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	m.handler.RegisterRoutes(router, authMw, adminMw)
	m.guardrailHandler.RegisterRoutes(router, authMw, adminMw)
	m.stepHandler.RegisterRoutes(router, authMw, adminMw)
	m.scoringHandler.RegisterRoutes(router, authMw, adminMw)
}

func (m *Module) SpecialistRepo() domain.SpecialistRepository {
	return m.specialistRepo
}
