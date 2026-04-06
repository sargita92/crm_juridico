package document

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/document/application"
	"github.com/sasrgita/crm-juridico/internal/document/infrastructure"
	documenthttp "github.com/sasrgita/crm-juridico/internal/document/interfaces/http"
)

type Module struct {
	handler *documenthttp.Handler
}

func NewModule(db *gorm.DB, specialistFinder application.SpecialistFinder) *Module {
	docRepo := infrastructure.NewGormDocumentRepository(db)
	specDocRepo := infrastructure.NewGormSpecialistDocumentRepository(db)

	uploadUC := application.NewUploadDocumentUseCase(docRepo)
	listUC := application.NewListDocumentsUseCase(docRepo, specDocRepo)
	getUC := application.NewGetDocumentUseCase(docRepo, specDocRepo)
	deleteUC := application.NewDeleteDocumentUseCase(docRepo, specDocRepo)
	associateUC := application.NewAssociateDocumentUseCase(specialistFinder, docRepo, specDocRepo)
	dissociateUC := application.NewDissociateDocumentUseCase(specDocRepo)
	listSpecDocsUC := application.NewListSpecialistDocumentsUseCase(specDocRepo, docRepo)
	listAvailableUC := application.NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	handler := documenthttp.NewHandler(
		uploadUC, listUC, getUC, deleteUC,
		associateUC, dissociateUC, listSpecDocsUC, listAvailableUC,
	)

	return &Module{handler: handler}
}

func (m *Module) Name() string { return "document" }

func (m *Module) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	m.handler.RegisterRoutes(router, authMw, adminMw)
}
