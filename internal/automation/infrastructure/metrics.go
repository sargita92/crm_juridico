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

func init() {
	prometheus.MustRegister(ExecutionsTotal)
}
