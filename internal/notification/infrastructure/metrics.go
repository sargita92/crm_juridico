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

	// NotificationReadTotal counts mark-read operations broken down by type.
	//
	// Semantics: one increment per call, not per notification. The "type"
	// label value is "single" for MarkRead(id) and "all" for MarkAllRead —
	// the actual NotificationType is not emitted to avoid an extra DB read
	// on a hot UI path (mark-as-read is clicked frequently).
	NotificationReadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "read_total",
			Help:      "Total mark-read calls on notifications, by type (single|all).",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(NotificationsDeliveredTotal, SSEActiveStreams, SSEEventsEmittedTotal, NotificationReadTotal)
}
