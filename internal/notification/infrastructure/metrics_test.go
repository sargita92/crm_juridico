package infrastructure

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestMetrics_Registered(t *testing.T) {
	// Touch each metric so they appear in the registry's output.
	NotificationsDeliveredTotal.WithLabelValues("lead_assigned").Inc()
	SSEActiveStreams.Inc()
	SSEActiveStreams.Dec()
	SSEEventsEmittedTotal.WithLabelValues("delivered").Inc()

	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var names []string
	for _, m := range metrics {
		names = append(names, m.GetName())
	}
	joined := strings.Join(names, "\n")
	assert.Contains(t, joined, "crm_notifications_delivered_total")
	assert.Contains(t, joined, "crm_notifications_sse_active_streams")
	assert.Contains(t, joined, "crm_notifications_sse_events_emitted_total")
}
