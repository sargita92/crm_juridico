package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// ExecutionsTotal counts automation executions, broken down by type and outcome.
// type ∈ {move_funnel, auto_note, switch_specialist, detect_product, auto_message, expiration}
// outcome ∈ {success, error} — mirrors domain.ExecutionStatus
var ExecutionsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "automation",
		Name:      "executions_total",
		Help:      "Total automation executions, by type and outcome.",
	},
	[]string{"type", "outcome"},
)

// ExecutionDuration measures the latency of automation executor runs.
// Labels:
//   - type: one of {move_funnel, auto_note, switch_specialist, detect_product, auto_message, expiration}
//   - outcome: one of {success, error} — mirrors ExecutionsTotal
//
// Buckets cover fast in-process operations (10 ms) up to WhatsApp/HTTP fan-out
// latencies (10 s).
var ExecutionDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "crm",
		Subsystem: "automation",
		Name:      "execution_duration_seconds",
		Help:      "Duration of automation executor runs, by type and outcome.",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
	},
	[]string{"type", "outcome"},
)

func init() {
	prometheus.MustRegister(ExecutionsTotal, ExecutionDuration)
}
