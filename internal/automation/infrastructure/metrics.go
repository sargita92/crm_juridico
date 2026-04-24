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

// RateLimitedTotal counts automation executions dropped because a rate limit
// was exceeded, broken down by type.
//
// As of F18 the automation module has NO active rate-limiting drop path —
// domain.RateLimitCounter / gorm_rate_limit_repo track volume but no
// executor consults them to reject work. This counter is registered now so
// dashboards and alert rules can reference the name without a schema change
// later; it stays at zero until the first call site is added (candidate:
// per-tenant/per-specialist limits on auto_message). When that happens, call
// RateLimitedTotal.WithLabelValues("<type>").Inc() on the drop path.
var RateLimitedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "automation",
		Name:      "rate_limited_total",
		Help:      "Total automation executions dropped because a rate limit was exceeded, by type.",
	},
	[]string{"type"},
)

func init() {
	prometheus.MustRegister(ExecutionsTotal, ExecutionDuration, RateLimitedTotal)
}
