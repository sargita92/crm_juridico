package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
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
	operators   application.OperatorLister
	log         *zap.Logger
}

func NewHandler(
	tenantUC *application.GetTenantDashboard,
	adminUC *application.GetAdminDashboard,
	userTenants authdomain.UserTenantRepository,
	operators application.OperatorLister,
	log *zap.Logger,
) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		tenantUC:    tenantUC,
		adminUC:     adminUC,
		userTenants: userTenants,
		operators:   operators,
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

	// F25 — seletor de operador: só o owner monta a lista e pode drillar num
	// usuário específico via ?user=<id>. Não-owner ignora o parâmetro e fica
	// sempre travado no próprio escopo.
	var (
		operators      []domain.Operator
		selectedUserID string
		viewUserID     *string
	)
	if isOwner {
		operators = h.listOperators(ctx, tenantID)
		if q := strings.TrimSpace(c.Query("user")); q != "" {
			if h.isValidOperator(ctx, q, tenantID) {
				id := q
				viewUserID = &id
				selectedUserID = q
			} else {
				h.log.Warn("dashboard tenant: invalid user selection ignored",
					zap.String("tenant_id", tenantID),
					zap.String("requested_user", q))
			}
		}
	}

	stats, err := h.tenantUC.Execute(ctx, application.TenantInput{
		TenantID:   tenantID,
		UserID:     claims.UserID,
		IsOwner:    isOwner,
		ViewUserID: viewUserID,
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
		"Stats":          vm,
		"TenantID":       tenantID,
		"Operators":      operators,
		"SelectedUserID": selectedUserID,
		"CanSelectUser":  isOwner,
	})

	h.log.Info("dashboard_rendered",
		zap.String("scope", scope),
		zap.String("tenant_id", tenantID),
		zap.String("user_id", claims.UserID),
		zap.Bool("scope_is_user", stats.ScopeIsUser),
		zap.String("viewed_user_id", selectedUserID),
		zap.Duration("took", time.Since(start)))
}

// listOperators devolve os operadores do tenant para o seletor; em caso de erro
// loga e devolve lista vazia (o dashboard ainda renderiza, só sem seletor populado).
func (h *Handler) listOperators(ctx context.Context, tenantID string) []domain.Operator {
	ops, err := h.operators.Operators(ctx, tenantID)
	if err != nil {
		h.log.Warn("dashboard tenant: list operators",
			zap.Error(err), zap.String("tenant_id", tenantID))
		return nil
	}
	return ops
}

// isValidOperator confirma que userID é um operador (não-owner) membro do tenant.
// Falso para não-membro, owner ou erro — nesses casos o owner cai em consolidado.
func (h *Handler) isValidOperator(ctx context.Context, userID, tenantID string) bool {
	ut, err := h.userTenants.FindByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		h.log.Warn("dashboard tenant: validate selected user",
			zap.Error(err), zap.String("tenant_id", tenantID))
		return false
	}
	return ut != nil && !ut.IsOwner
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
