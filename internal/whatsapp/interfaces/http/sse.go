package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

func (h *Handler) HandleSSE(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	if tenantID == "" {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	eventCh, cleanup := h.eventBus.Subscribe(tenantID)
	defer cleanup()

	// Initial keepalive to establish connection
	c.Writer.WriteString(": keepalive\n\n")
	c.Writer.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return false
			}
			h.log.Info("SSE sending event",
				zap.String("tenant_id", tenantID),
				zap.String("event_type", string(event.Type)),
			)
			fmt.Fprintf(w, "event: %s\ndata: {}\n\n", event.Type)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
