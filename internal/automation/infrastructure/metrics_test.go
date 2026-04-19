package infrastructure

import (
	"testing"

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
	}
}
