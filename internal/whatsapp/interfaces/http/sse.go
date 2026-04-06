package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

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

	c.Stream(func(w io.Writer) bool {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "event: %s\n", event.Type)
			fmt.Fprintf(w, "data: %v\n\n", event.Payload)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
