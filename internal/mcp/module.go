package mcp

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/mcp/application"
	"github.com/sasrgita/crm-juridico/internal/mcp/infrastructure"
	mcphttp "github.com/sasrgita/crm-juridico/internal/mcp/interfaces/http"
)

type Module struct {
	handler *mcphttp.Handler
}

func NewModule(db *gorm.DB, specialistFinder application.SpecialistFinder) *Module {
	mcpRepo := infrastructure.NewGormMcpServerRepository(db)
	specMcpRepo := infrastructure.NewGormSpecialistMcpRepository(db)

	createUC := application.NewCreateMcpUseCase(mcpRepo)
	listUC := application.NewListMcpsUseCase(mcpRepo, specMcpRepo)
	getUC := application.NewGetMcpUseCase(mcpRepo, specMcpRepo)
	updateUC := application.NewUpdateMcpUseCase(mcpRepo)
	deleteUC := application.NewDeleteMcpUseCase(mcpRepo, specMcpRepo)
	associateUC := application.NewAssociateMcpUseCase(specialistFinder, mcpRepo, specMcpRepo)
	dissociateUC := application.NewDissociateMcpUseCase(specMcpRepo)
	listSpecMcpsUC := application.NewListSpecialistMcpsUseCase(specMcpRepo, mcpRepo)
	listAvailableUC := application.NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	handler := mcphttp.NewHandler(
		createUC, listUC, getUC, updateUC, deleteUC,
		associateUC, dissociateUC, listSpecMcpsUC, listAvailableUC,
	)

	return &Module{handler: handler}
}

func (m *Module) Name() string { return "mcp" }

func (m *Module) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	m.handler.RegisterRoutes(router, authMw, adminMw)
}
