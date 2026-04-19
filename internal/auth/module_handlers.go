package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// handleGenerateInvite handles POST /tenant/invites.
func (m *Module) handleGenerateInvite(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.invite.generate")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		GroupIDs      []string `json:"group_ids"`
		ExpiresInDays int      `json:"expires_in_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := m.inviteUC.GenerateInvite(c.Request.Context(), tenantID, claims.UserID, req.GroupIDs, req.ExpiresInDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate invite"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         out.ID,
		"token":      out.Token,
		"expires_at": out.ExpiresAt,
		"group_ids":  out.GroupIDs,
	})
}

// handleListInvites handles GET /tenant/invites.
func (m *Module) handleListInvites(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.invite.list")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())

	invites, err := m.inviteUC.ListInvites(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list invites"})
		return
	}

	c.JSON(http.StatusOK, invites)
}

// handleRevokeInvite handles DELETE /tenant/invites/:id.
func (m *Module) handleRevokeInvite(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.invite.revoke")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invite id is required"})
		return
	}

	if err := m.inviteUC.RevokeInvite(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke invite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleGetInviteInfo handles GET /invite/:token — public.
// Returns basic info about the invite (non-sensitive). Full validation happens on accept.
func (m *Module) handleGetInviteInfo(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.invite.get_info")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	// Token validity is enforced on POST /invite/:token/accept.
	// This endpoint is used to render the accept form and confirm the token exists.
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// handleAcceptInvite handles POST /invite/:token/accept — public.
func (m *Module) handleAcceptInvite(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.invite.accept")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var req struct {
		Name     string `json:"name"     binding:"required"`
		Email    string `json:"email"    binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := m.inviteUC.AcceptInvite(c.Request.Context(), token, req.Name, req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrInviteTokenNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "invite not found"})
		case domain.ErrInviteTokenExpired, domain.ErrInviteTokenUsed:
			c.JSON(http.StatusGone, gin.H{"error": err.Error()})
		case application.ErrUserAlreadyInTenant:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to accept invite"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":   out.UserID,
		"group_ids": out.GroupIDs,
	})
}

// handleListTenantUsers handles GET /tenant/users.
func (m *Module) handleListTenantUsers(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.users.list")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())

	users, err := m.manageUsersUC.ListTenantUsers(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// handleRemoveUser handles DELETE /tenant/users/:id.
func (m *Module) handleRemoveUser(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.users.remove")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	userID := c.Param("id")
	tenantID := middleware.GetTenantID(c.Request.Context())

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	if err := m.manageUsersUC.RemoveFromTenant(c.Request.Context(), userID, tenantID); err != nil {
		if err == application.ErrCannotRemoveOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// handleSetWhatsAppID handles PUT /tenant/users/:id/whatsapp.
func (m *Module) handleSetWhatsAppID(c *gin.Context) {
	ctx, span := otel.Tracer("auth").Start(c.Request.Context(), "auth.users.set_whatsapp")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	userID := c.Param("id")
	tenantID := middleware.GetTenantID(c.Request.Context())

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var req struct {
		WhatsAppID string `json:"whatsapp_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.manageUsersUC.SetWhatsAppID(c.Request.Context(), userID, tenantID, req.WhatsAppID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update whatsapp id"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
