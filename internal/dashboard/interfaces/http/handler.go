package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Handler expõe as rotas do dashboard tenant e admin. Os bodies completos
// (com view models, fragments HTMX e templates reais) chegam nas Tasks 12 e 13.
type Handler struct {
	tenantUC    *application.GetTenantDashboard
	adminUC     *application.GetAdminDashboard
	userTenants authdomain.UserTenantRepository
	log         *zap.Logger
}

func NewHandler(
	tenantUC *application.GetTenantDashboard,
	adminUC *application.GetAdminDashboard,
	userTenants authdomain.UserTenantRepository,
	log *zap.Logger,
) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		tenantUC:    tenantUC,
		adminUC:     adminUC,
		userTenants: userTenants,
		log:         log,
	}
}

// RegisterRoutes liga as 4 rotas do F19. Bodies de Tasks 12/13 substituem os stubs.
func (h *Handler) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	// Tenant: precisa de Auth + Tenant context
	tenantGroup := router.Group("/dashboard")
	tenantGroup.Use(mw.Auth, mw.Tenant)
	tenantGroup.GET("", h.tenantPage)
	tenantGroup.GET("/content", h.tenantFragment)

	// Admin: Auth + Admin
	adminGroup := router.Group("/admin/dashboard")
	adminGroup.Use(mw.Auth, mw.Admin)
	adminGroup.GET("", h.adminPage)
	adminGroup.GET("/content", h.adminFragment)
}

// Stubs — Task 12/13 substitui.
func (h *Handler) tenantPage(c *gin.Context) {
	c.String(http.StatusOK, "tenant dashboard placeholder (Task 12)")
}
func (h *Handler) tenantFragment(c *gin.Context) {
	c.String(http.StatusOK, "tenant fragment placeholder (Task 12)")
}
func (h *Handler) adminPage(c *gin.Context) {
	c.String(http.StatusOK, "admin dashboard placeholder (Task 13)")
}
func (h *Handler) adminFragment(c *gin.Context) {
	c.String(http.StatusOK, "admin fragment placeholder (Task 13)")
}
