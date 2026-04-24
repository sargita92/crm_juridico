package infrastructure

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestExecutionsTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(ExecutionsTotal.WithLabelValues("auto_message", "success"))
	ExecutionsTotal.WithLabelValues("auto_message", "success").Inc()
	after := testutil.ToFloat64(ExecutionsTotal.WithLabelValues("auto_message", "success"))
	assert.Equal(t, before+1, after)
}

func TestExecutionsTotal_AcceptsAllExecutorTypes(t *testing.T) {
	types := []string{"move_funnel", "auto_note", "switch_specialist", "detect_product", "auto_message", "expiration"}
	for _, tp := range types {
		ExecutionsTotal.WithLabelValues(tp, "success").Inc()
		ExecutionsTotal.WithLabelValues(tp, "error").Inc()
		// Assert that both success and error counters were incremented.
		successValue := testutil.ToFloat64(ExecutionsTotal.WithLabelValues(tp, "success"))
		errorValue := testutil.ToFloat64(ExecutionsTotal.WithLabelValues(tp, "error"))
		assert.GreaterOrEqual(t, successValue, float64(1), "success counter should be incremented for %s", tp)
		assert.GreaterOrEqual(t, errorValue, float64(1), "error counter should be incremented for %s", tp)
	}
}

func TestExecutionsTotal_Registered(t *testing.T) {
	// Touch the metric so it appears in the registry's output.
	ExecutionsTotal.WithLabelValues("auto_message", "success").Inc()

	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var names []string
	for _, m := range metrics {
		names = append(names, m.GetName())
	}
	joined := strings.Join(names, "\n")
	assert.Contains(t, joined, "crm_automation_executions_total")
}

func TestExecutionDuration_ObservesAcrossTypes(t *testing.T) {
	types := []string{"move_funnel", "auto_note", "switch_specialist", "detect_product", "auto_message", "expiration"}
	for _, tp := range types {
		ExecutionDuration.WithLabelValues(tp, "success").Observe(0.05)
		ExecutionDuration.WithLabelValues(tp, "error").Observe(0.1)
	}
	// The vector should expose at least one series per (type, outcome) pair we
	// observed (6 types × 2 outcomes = 12). Other outcomes already observed in
	// other tests may push the count higher, so just require the lower bound.
	assert.GreaterOrEqual(t, testutil.CollectAndCount(ExecutionDuration), 12)
}

func TestExecutionDuration_Registered(t *testing.T) {
	// Touch the histogram so the family appears in the registry.
	ExecutionDuration.WithLabelValues("auto_message", "success").Observe(0.01)

	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var names []string
	for _, m := range metrics {
		names = append(names, m.GetName())
	}
	joined := strings.Join(names, "\n")
	assert.Contains(t, joined, "crm_automation_execution_duration_seconds")
}
