package files

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/files/application"
	"github.com/sasrgita/crm-juridico/internal/files/domain"
	"github.com/sasrgita/crm-juridico/internal/files/infrastructure"
	fileshttp "github.com/sasrgita/crm-juridico/internal/files/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type Module struct {
	repo       domain.FileRepository
	storage    domain.Storage
	storeUC    *application.StoreFileUseCase
	listUC     *application.ListFilesUseCase
	getUC      *application.GetFileUseCase
	downloadUC *application.DownloadFileUseCase
	summaryUC  *application.LeadFilesSummaryUseCase
	adapter    *application.WhatsAppFileAdapter
	handler    *fileshttp.Handler
	log        *zap.Logger
}

// NewModule wires the files module. `storageRoot` is the directory where
// LocalDiskStorage persists bytes (tenant-scoped subpaths). `maxBytes` caps
// inbound captures; pass 0 for the default (50 MB). `leadLookup` is the
// funnel module's adapter to correlate files with leads; if nil, files will
// be persisted with lead_id=NULL.
func NewModule(
	db *gorm.DB,
	log *zap.Logger,
	leadLookup domain.LeadLookup,
	storageRoot string,
	maxBytes int64,
) (*Module, error) {
	if log == nil {
		log = zap.NewNop()
	}
	storage, err := infrastructure.NewLocalDiskStorage(storageRoot)
	if err != nil {
		return nil, fmt.Errorf("files module: storage init: %w", err)
	}
	repo := infrastructure.NewGormFileRepository(db)

	storeUC := application.NewStoreFileUseCase(repo, storage, leadLookup, maxBytes)
	listUC := application.NewListFilesUseCase(repo)
	getUC := application.NewGetFileUseCase(repo)
	downloadUC := application.NewDownloadFileUseCase(repo, storage)
	summaryUC := application.NewLeadFilesSummaryUseCase(repo)
	adapter := application.NewWhatsAppFileAdapter(storeUC)

	handler := fileshttp.NewHandler(listUC, getUC, downloadUC, summaryUC, log)

	return &Module{
		repo:       repo,
		storage:    storage,
		storeUC:    storeUC,
		listUC:     listUC,
		getUC:      getUC,
		downloadUC: downloadUC,
		summaryUC:  summaryUC,
		adapter:    adapter,
		handler:    handler,
		log:        log,
	}, nil
}

func (m *Module) Name() string { return "files" }

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
}

// FileStorer exposes the whatsapp-facing port.
func (m *Module) FileStorer() whatsappdomain.FileStorer { return m.adapter }

// LeadFilesSummary exposes the use case used by the funnel module to render
// the file section in the lead detail modal.
func (m *Module) LeadFilesSummary() *application.LeadFilesSummaryUseCase { return m.summaryUC }

// ListFilesUC exposes the list use case (used by the HTTP handler in step 5).
func (m *Module) ListFilesUC() *application.ListFilesUseCase { return m.listUC }

// GetFileUC exposes the get use case.
func (m *Module) GetFileUC() *application.GetFileUseCase { return m.getUC }

// DownloadFileUC exposes the download use case.
func (m *Module) DownloadFileUC() *application.DownloadFileUseCase { return m.downloadUC }
