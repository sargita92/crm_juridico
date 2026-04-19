package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
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

func (h *Handler) tenantPage(c *gin.Context) {
	h.renderTenant(c, "tenant/dashboard/page.html")
}

func (h *Handler) tenantFragment(c *gin.Context) {
	h.renderTenant(c, "tenant/dashboard/content.html")
}

func (h *Handler) renderTenant(c *gin.Context, tmpl string) {
	start := time.Now()
	scope := "tenant"
	outcome := "success"
	defer func() {
		infrastructure.RenderDuration.WithLabelValues(scope).Observe(time.Since(start).Seconds())
		infrastructure.LoadTotal.WithLabelValues(scope, outcome).Inc()
	}()

	ctx, span := httpTracer.Start(c.Request.Context(), "dashboard.http.tenant")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	claims := middleware.GetClaims(ctx)
	if claims == nil {
		outcome = "error"
		c.String(http.StatusUnauthorized, "no claims")
		return
	}
	tenantID := middleware.GetTenantID(ctx)
	if tenantID == "" {
		outcome = "error"
		c.String(http.StatusBadRequest, "no tenant context")
		return
	}
	isOwner, err := h.userTenants.IsOwner(ctx, claims.UserID, tenantID)
	if err != nil {
		outcome = "error"
		h.log.Error("dashboard tenant: check owner",
			zap.Error(err),
			zap.String("tenant_id", tenantID),
			zap.String("user_id", claims.UserID))
		c.String(http.StatusInternalServerError, "erro interno")
		return
	}
	// Admins de plataforma também são tratados como owner para o dashboard do tenant.
	if claims.Role == authdomain.UserRoleAdmin {
		isOwner = true
	}
	stats, err := h.tenantUC.Execute(ctx, application.TenantInput{
		TenantID: tenantID,
		UserID:   claims.UserID,
		IsOwner:  isOwner,
	})
	if err != nil {
		outcome = "error"
		h.log.Error("dashboard tenant: execute",
			zap.Error(err),
			zap.String("tenant_id", tenantID),
			zap.String("user_id", claims.UserID))
		c.String(http.StatusInternalServerError, "erro ao carregar dashboard")
		return
	}
	vm := ToTenantView(stats)
	c.HTML(http.StatusOK, tmpl, gin.H{
		"Stats":    vm,
		"TenantID": tenantID,
	})

	h.log.Info("dashboard_rendered",
		zap.String("scope", scope),
		zap.String("tenant_id", tenantID),
		zap.String("user_id", claims.UserID),
		zap.Bool("scope_is_user", !isOwner),
		zap.Duration("took", time.Since(start)))
}

func (h *Handler) adminPage(c *gin.Context) {
	h.renderAdmin(c, "admin/dashboard/page.html")
}

func (h *Handler) adminFragment(c *gin.Context) {
	h.renderAdmin(c, "admin/dashboard/content.html")
}

func (h *Handler) renderAdmin(c *gin.Context, tmpl string) {
	start := time.Now()
	scope := "admin"
	outcome := "success"
	defer func() {
		infrastructure.RenderDuration.WithLabelValues(scope).Observe(time.Since(start).Seconds())
		infrastructure.LoadTotal.WithLabelValues(scope, outcome).Inc()
	}()

	ctx, span := httpTracer.Start(c.Request.Context(), "dashboard.http.admin")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	claims := middleware.GetClaims(ctx)

	stats, err := h.adminUC.Execute(ctx)
	if err != nil {
		outcome = "error"
		userID := ""
		if claims != nil {
			userID = claims.UserID
		}
		h.log.Error("dashboard admin: execute",
			zap.Error(err),
			zap.String("user_id", userID))
		c.String(http.StatusInternalServerError, "erro ao carregar dashboard")
		return
	}
	vm := ToAdminView(stats)
	c.HTML(http.StatusOK, tmpl, gin.H{"Stats": vm})

	if claims != nil {
		h.log.Info("dashboard_rendered",
			zap.String("scope", scope),
			zap.String("user_id", claims.UserID),
			zap.Duration("took", time.Since(start)))
	}
}
