package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// PageHandler serves HTML pages and fragments for the notification UI.
type PageHandler struct {
	listUC     *application.ListNotificationsUseCase
	markReadUC *application.MarkReadUseCase
	log        *zap.Logger
}

// NewPageHandler constructs a PageHandler.
func NewPageHandler(
	listUC *application.ListNotificationsUseCase,
	markReadUC *application.MarkReadUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{listUC: listUC, markReadUC: markReadUC, log: log}
}

// RenderBadge returns the unread-count badge fragment. Zero count renders
// an empty/hidden badge so the UI doesn't show a 0 to the user.
func (h *PageHandler) RenderBadge(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	count, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
	if err != nil {
		h.log.Error("page: count unread", zap.Error(err))
		count = 0
	}

	c.HTML(http.StatusOK, "partials/notification_badge.html", gin.H{"Count": count})
}

// RenderDropdown returns the dropdown body with up to 10 most-recent
// notifications (read + unread) for the authenticated user.
func (h *PageHandler) RenderDropdown(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	items, err := h.listUC.Execute(c.Request.Context(), tenantID, claims.UserID, false, 10, 0)
	if err != nil {
		h.log.Error("page: list dropdown", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	unread, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
	if err != nil {
		h.log.Warn("page: count unread for dropdown", zap.Error(err))
		unread = 0
	}

	c.HTML(http.StatusOK, "partials/notification_dropdown.html", gin.H{
		"Items":       items,
		"UnreadCount": unread,
	})
}
