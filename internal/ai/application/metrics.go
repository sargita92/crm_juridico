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

	aiResetCommandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "reset_commands_total",
			Help:      "Total number of conversation resets, by source.",
		},
		[]string{"tenant_id", "specialist_id", "source"},
	)

	aiPlaygroundMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "playground_messages_total",
			Help:      "Total number of messages injected via the dev playground.",
		},
		[]string{"tenant_id"},
	)

	aiToolCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tool_calls_total",
			Help:      "Total number of tool calls executed",
		},
		[]string{"tenant_id", "specialist_id", "tool_name", "status"},
	)

	aiToolCallDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tool_call_duration_seconds",
			Help:      "Duration of tool call execution",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"tenant_id", "tool_name"},
	)

	aiToolLoopIterations = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tool_loop_iterations",
			Help:      "Number of iterations in the tool calling loop",
			Buckets:   []float64{1, 2, 3, 4, 5},
		},
		[]string{"tenant_id", "specialist_id"},
	)

	aiToolResultTruncatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "ai",
			Name:      "tool_result_truncated_total",
			Help:      "Total number of tool results that were truncated",
		},
		[]string{"tenant_id", "tool_name"},
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
		aiResetCommandsTotal,
		aiPlaygroundMessagesTotal,
		aiToolCallsTotal,
		aiToolCallDurationSeconds,
		aiToolLoopIterations,
		aiToolResultTruncatedTotal,
	)
}
