package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
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

// RenderList returns the notifications list fragment (tab content + pagination)
// for the authenticated user. Query: ?filter=unread|all&limit=20&offset=0
func (h *PageHandler) RenderList(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	filter := c.DefaultQuery("filter", "unread")
	onlyUnread := filter == "unread"

	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Fetch one extra to know if there's a next page.
	items, err := h.listUC.Execute(c.Request.Context(), tenantID, claims.UserID, onlyUnread, limit+1, offset)
	if err != nil {
		h.log.Error("page: list", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	c.HTML(http.StatusOK, "notification/list_items.html", gin.H{
		"Items":      items,
		"Filter":     filter,
		"Limit":      limit,
		"Offset":     offset,
		"HasMore":    hasMore,
		"HasPrev":    offset > 0,
		"NextOffset": offset + limit,
		"PrevOffset": maxZero(offset - limit),
	})
}

func maxZero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// RenderPage returns the full /tenant/notifications page with the unread tab
// selected by default. The tab content is loaded on demand via RenderList.
func (h *PageHandler) RenderPage(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	filter := c.DefaultQuery("filter", "unread")
	if filter != "unread" && filter != "all" {
		filter = "unread"
	}

	unread, _ := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)

	c.HTML(http.StatusOK, "notification/list.html", gin.H{
		"Filter":      filter,
		"UnreadCount": unread,
		"ActiveNav":   "", // no sidebar item highlights on this page
	})
}
