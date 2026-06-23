package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
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
