package application

import "github.com/prometheus/client_golang/prometheus"

var (
	aiRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "requests_total",
			Help:      "Total number of AI requests.",
		},
		[]string{"tenant_id", "specialist_id", "provider", "model", "status"},
	)

	aiRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "request_duration_seconds",
			Help:      "Duration of AI requests in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"tenant_id", "provider", "model"},
	)

	aiTokensPromptTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tokens_prompt_total",
			Help:      "Total number of prompt tokens sent to AI providers.",
		},
		[]string{"tenant_id", "specialist_id", "provider"},
	)

	aiTokensCompletionTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tokens_completion_total",
			Help:      "Total number of completion tokens received from AI providers.",
		},
		[]string{"tenant_id", "specialist_id", "provider"},
	)

	aiGuardrailViolationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "guardrail_violations_total",
			Help:      "Total number of guardrail violations detected in AI responses.",
		},
		[]string{"tenant_id", "specialist_id", "guardrail_type"},
	)

	aiStepsCompletedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "steps_completed_total",
			Help:      "Total number of conversation steps completed.",
		},
		[]string{"tenant_id", "specialist_id"},
	)

	aiLeadsQualifiedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "leads_qualified_total",
			Help:      "Total number of leads fully qualified by the AI.",
		},
		[]string{"tenant_id", "specialist_id"},
	)

	aiLeadsDisqualifiedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "leads_disqualified_total",
			Help:      "Total number of leads disqualified by the AI.",
		},
		[]string{"tenant_id", "specialist_id"},
	)

	aiHandoffsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "handoffs_total",
			Help:      "Total number of conversation handoffs.",
		},
		[]string{"tenant_id", "direction"},
	)
)

func init() {
	prometheus.MustRegister(
		aiRequestsTotal,
		aiRequestDuration,
		aiTokensPromptTotal,
		aiTokensCompletionTotal,
		aiGuardrailViolationsTotal,
		aiStepsCompletedTotal,
		aiLeadsQualifiedTotal,
		aiLeadsDisqualifiedTotal,
		aiHandoffsTotal,
	)
}
