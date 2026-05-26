package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	notifinfra "github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
	events "github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// Handler holds all notification use cases and the event bus for SSE.
type Handler struct {
	notifySvc     *application.NotifyService
	listUC        *application.ListNotificationsUseCase
	markReadUC    *application.MarkReadUseCase
	preferencesUC *application.ManagePreferencesUseCase
	eventBus      events.EventBus
	renderer      *ToastRenderer
	log           *zap.Logger
}

// NewHandler builds a notification Handler.
func NewHandler(
	notifySvc *application.NotifyService,
	listUC *application.ListNotificationsUseCase,
	markReadUC *application.MarkReadUseCase,
	preferencesUC *application.ManagePreferencesUseCase,
	eventBus events.EventBus,
	renderer *ToastRenderer,
	log *zap.Logger,
) *Handler {
	return &Handler{
		notifySvc:     notifySvc,
		listUC:        listUC,
		markReadUC:    markReadUC,
		preferencesUC: preferencesUC,
		eventBus:      eventBus,
		renderer:      renderer,
		log:           log,
	}
}

// SetRenderer allows late-binding injection of the renderer after the router's
// template set has been parsed. Required because the module is constructed
// before `setupRouter` parses templates.
func (h *Handler) SetRenderer(r *ToastRenderer) { h.renderer = r }

// StreamNotifications opens a Server-Sent Events stream for the authenticated user.
// Each event is delivered as an HTML fragment: the toast markup plus an OOB
// swap that refreshes the unread-count badge.
func (h *Handler) StreamNotifications(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch, cleanup := h.eventBus.Subscribe(tenantID)
	defer cleanup()

	notifinfra.SSEActiveStreams.Inc()
	defer notifinfra.SSEActiveStreams.Dec()

	// Note: c.Stream is intentionally avoided here because gin v1.12.0's
	// c.Stream calls w.CloseNotify() unconditionally, which panics when the
	// underlying ResponseWriter is *httptest.ResponseRecorder (no CloseNotifier).
	// c.SSEvent itself does not call CloseNotify and is safe to use directly.
	flusher, canFlush := c.Writer.(http.Flusher)

	for {
		select {
		case event := <-ch:
			if event.Type != events.EventNotification {
				// Stream unificado (F26): encaminha eventos que não são notificação
				// (new-message, conversation-update, lead-*) como eventos SSE nomeados,
				// para que os consumidores htmx (hx-trigger="sse:<tipo>") disparem a
				// partir de uma única conexão por página.
				c.SSEvent(string(event.Type), "{}")
				if canFlush {
					flusher.Flush()
				}
				notifinfra.SSEEventsEmittedTotal.WithLabelValues("forwarded").Inc()
				continue
			}
			notif, ok := event.Payload.(*domain.Notification)
			if !ok {
				notifinfra.SSEEventsEmittedTotal.WithLabelValues("skipped").Inc()
				continue
			}
			if notif.UserID != claims.UserID {
				notifinfra.SSEEventsEmittedTotal.WithLabelValues("skipped").Inc()
				continue
			}
			if h.renderer == nil {
				h.log.Warn("sse: renderer not set; skipping event")
				notifinfra.SSEEventsEmittedTotal.WithLabelValues("skipped").Inc()
				continue
			}

			ctx, span := otel.Tracer("notification").Start(c.Request.Context(), "notification.stream.emit")
			count, err := h.listUC.CountUnread(ctx, tenantID, claims.UserID)
			if err != nil {
				h.log.Warn("sse count unread failed", zap.Error(err))
				count = 0
			}

			fragment, err := h.renderer.Render(notif, count)
			if err != nil {
				h.log.Error("sse render failed", zap.Error(err))
				notifinfra.SSEEventsEmittedTotal.WithLabelValues("render_error").Inc()
				span.End()
				continue
			}

			// c.SSEvent uses gin-contrib/sse which correctly prefixes every
			// line of a multi-line data value with "data:", per the SSE spec.
			c.SSEvent("notification", fragment)
			if canFlush {
				flusher.Flush()
			}
			notifinfra.SSEEventsEmittedTotal.WithLabelValues("delivered").Inc()
			span.End()

		case <-c.Request.Context().Done():
			return
		}
	}
}

// ListNotifications returns paginated notifications for the authenticated user.
func (h *Handler) ListNotifications(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	onlyUnread := c.Query("unread") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	items, err := h.listUC.Execute(c.Request.Context(), tenantID, claims.UserID, onlyUnread, limit, offset)
	if err != nil {
		h.log.Error("list notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notifications"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// UnreadCount returns the count of unread notifications.
func (h *Handler) UnreadCount(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	count, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
	if err != nil {
		h.log.Error("count unread notifications", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count unread notifications"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

// MarkRead marks a single notification as read.
func (h *Handler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification id is required"})
		return
	}

	if err := h.markReadUC.MarkRead(c.Request.Context(), id); err != nil {
		if err == domain.ErrNotificationNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		h.log.Error("mark notification read", zap.Error(err), zap.String("id", id))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notification as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// MarkAllRead marks all notifications as read for the authenticated user.
func (h *Handler) MarkAllRead(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	if err := h.markReadUC.MarkAllRead(c.Request.Context(), tenantID, claims.UserID); err != nil {
		h.log.Error("mark all notifications read", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark all notifications as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetPreferences returns notification preferences for the authenticated user.
func (h *Handler) GetPreferences(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	prefs, err := h.preferencesUC.GetPreferences(c.Request.Context(), claims.UserID, tenantID)
	if err != nil {
		h.log.Error("get notification preferences", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get preferences"})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences creates or updates a notification preference.
func (h *Handler) UpdatePreferences(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		Channel string `json:"channel" binding:"required"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channel := domain.Channel(req.Channel)
	if err := h.preferencesUC.SetPreference(c.Request.Context(), claims.UserID, tenantID, channel, req.Enabled); err != nil {
		if err == domain.ErrInvalidChannel {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		h.log.Error("update notification preference", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update preference"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
