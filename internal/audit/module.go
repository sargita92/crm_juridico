package audit

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
	audithttp "github.com/sasrgita/crm-juridico/internal/audit/interfaces/http"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// Module e o composition root do dominio audit (F12).
//
// Estado atual (Step 8): expoe os UCs, o Publisher default, o repo e o
// Handler HTTP. Os handlers usam adapters injetados via AttachFilters
// (TenantLister + AdminUserLister) — opcionais; sem eles os dropdowns da
// UI ficam vazios mas a tabela continua funcional.
type Module struct {
	RegisterUC *application.RegisterAuditLogUseCase
	ListUC     *application.ListAuditLogsUseCase
	GetUC      *application.GetAuditLogUseCase
	Publisher  application.Publisher
	Repo       domain.Repository

	// handler e construido em NewModule sem listers; AttachFilters
	// reconstroi o handler com os listers preenchidos. Mantemos um unico
	// ponteiro de Handler para o composition root chamar RegisterRoutes
	// uma so vez (apos AttachFilters).
	handler *audithttp.Handler
}

// NewModule monta as dependencias do contexto audit.
//
// Wire-up minimo: repositorio Gorm + caso de uso de registro + publisher
// default (engole erro com WARN, decisao do design F12 secao 3.1).
//
// `logger` pode ser nil — substituido por Nop dentro do UC/Publisher.
func NewModule(db *gorm.DB, logger *zap.Logger) *Module {
	if logger == nil {
		logger = zap.NewNop()
	}

	repo := infrastructure.NewGormAuditLogRepository(db)
	registerUC := application.NewRegisterAuditLogUseCase(repo, logger)
	listUC := application.NewListAuditLogsUseCase(repo, logger)
	getUC := application.NewGetAuditLogUseCase(repo, logger)
	publisher := application.NewPublisher(registerUC, logger)
	handler := audithttp.NewHandler(listUC, getUC, nil, nil)

	return &Module{
		RegisterUC: registerUC,
		ListUC:     listUC,
		GetUC:      getUC,
		Publisher:  publisher,
		Repo:       repo,
		handler:    handler,
	}
}

// AttachFilters injeta os adapters que populam os dropdowns de filtro da
// Tela 1 (filtro por tenant e por usuario admin). Pode ser chamado uma
// unica vez depois do NewModule, antes do RegisterRoutes — caso
// contrario o handler segue funcionando, mas com dropdowns vazios.
//
// Argumentos sao interfaces para permitir mocks em testes (nao acopla a
// `internal/audit/infrastructure`).
func (m *Module) AttachFilters(tenantLister domain.TenantLister, adminUserLister domain.AdminUserLister) {
	m.handler = audithttp.NewHandler(m.ListUC, m.GetUC, tenantLister, adminUserLister)
}

// RegisterRoutes monta /admin/logs e /admin/logs/:id no router.
//
// `tokenProvider` e usado pelo middleware AdminPageAuth. O middleware
// AdminOr404 e aplicado dentro de Handler.RegisterRoutes — composition
// root nao precisa montar a cadeia.
func (m *Module) RegisterRoutes(router *gin.Engine, tokenProvider authdomain.TokenProvider) {
	m.handler.RegisterRoutes(router, tokenProvider)
}
