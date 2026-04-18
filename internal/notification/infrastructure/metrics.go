package infrastructure

import "github.com/prometheus/client_golang/prometheus"

var (
	// NotificationsDeliveredTotal counts successfully persisted notifications,
	// broken down by type.
	NotificationsDeliveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "delivered_total",
			Help:      "Total notifications persisted, by type.",
		},
		[]string{"type"},
	)

	// SSEActiveStreams tracks the number of currently open SSE streams.
	SSEActiveStreams = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "sse_active_streams",
			Help:      "Number of currently open SSE notification streams.",
		},
	)

	// SSEEventsEmittedTotal counts SSE events pushed to clients, by outcome.
	SSEEventsEmittedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "sse_events_emitted_total",
			Help:      "Total SSE events emitted, by outcome (delivered, skipped, render_error).",
		},
		[]string{"outcome"},
	)
)

func init() {
	prometheus.MustRegister(NotificationsDeliveredTotal, SSEActiveStreams, SSEEventsEmittedTotal)
}
