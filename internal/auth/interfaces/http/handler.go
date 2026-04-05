package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

type Handler struct {
	loginUC        *application.LoginUseCase
	selectTenantUC *application.SelectTenantUseCase
	listTenantsUC  *application.ListUserTenantsUseCase
	secureCookie   bool
}

func NewHandler(
	loginUC *application.LoginUseCase,
	selectTenantUC *application.SelectTenantUseCase,
	listTenantsUC *application.ListUserTenantsUseCase,
	secureCookie bool,
) *Handler {
	return &Handler{
		loginUC:        loginUC,
		selectTenantUC: selectTenantUC,
		listTenantsUC:  listTenantsUC,
		secureCookie:   secureCookie,
	}
}

func (h *Handler) setTokenCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", token, maxAge, "/", "", h.secureCookie, true)
}

func (h *Handler) RenderLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/login.html", gin.H{"Error": ""})
}

func (h *Handler) HandleLogin(c *gin.Context) {
	email := c.PostForm("email")
	password := c.PostForm("password")

	if email == "" || password == "" {
		h.renderLoginError(c)
		return
	}

	output, err := h.loginUC.Execute(c.Request.Context(), application.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		h.renderLoginError(c)
		return
	}

	h.setTokenCookie(c, output.Token, 86400)

	if len(output.Tenants) == 1 {
		c.Header("HX-Redirect", "/dashboard")
	} else {
		c.Header("HX-Redirect", "/select-tenant")
	}
	c.Status(http.StatusOK)
}

func (h *Handler) RenderSelectTenant(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	if claims == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	tenants, err := h.listTenantsUC.Execute(c.Request.Context(), claims.UserID, claims.Role)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "auth/select_tenant.html", gin.H{"Tenants": nil})
		return
	}

	c.HTML(http.StatusOK, "auth/select_tenant.html", gin.H{"Tenants": tenants})
}

func (h *Handler) HandleSelectTenant(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	tenantID := c.PostForm("tenant_id")
	if tenantID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	output, err := h.selectTenantUC.Execute(c.Request.Context(), application.SelectTenantInput{
		UserID:   claims.UserID,
		TenantID: tenantID,
		Role:     claims.Role,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	h.setTokenCookie(c, output.Token, 86400)
	c.Header("HX-Redirect", "/dashboard")
	c.Status(http.StatusOK)
}

func (h *Handler) HandleLogout(c *gin.Context) {
	h.setTokenCookie(c, "", -1)
	c.Header("HX-Redirect", "/login")
	c.Status(http.StatusOK)
}

func (h *Handler) RenderDashboard(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	userName := ""
	if claims != nil {
		userName = claims.UserID // placeholder until we fetch the name
	}
	c.HTML(http.StatusOK, "auth/dashboard.html", gin.H{
		"UserName": userName,
	})
}

func (h *Handler) renderLoginError(c *gin.Context) {
	// HTMX requests get the card partial (for hx-swap="outerHTML")
	// Non-HTMX requests get the full page
	tmpl := "auth/login.html"
	if c.GetHeader("HX-Request") == "true" {
		tmpl = "auth/login_card.html"
	}
	c.HTML(http.StatusOK, tmpl, gin.H{"Error": "Email ou senha inválidos"})
}

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	// Public routes
	router.GET("/login", h.RenderLogin)
	router.POST("/login", h.HandleLogin)

	// Authenticated routes (no tenant required)
	authGroup := router.Group("/")
	authGroup.Use(authMw)
	authGroup.GET("/select-tenant", h.RenderSelectTenant)
	authGroup.POST("/select-tenant", h.HandleSelectTenant)
	authGroup.POST("/logout", h.HandleLogout)

	// Authenticated + tenant required
	tenantGroup := router.Group("/")
	tenantGroup.Use(authMw, tenantMw)
	tenantGroup.GET("/dashboard", h.RenderDashboard)
}
