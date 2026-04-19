package infrastructure

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestChangesTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(ChangesTotal.WithLabelValues("group", "updated"))
	ChangesTotal.WithLabelValues("group", "updated").Inc()
	after := testutil.ToFloat64(ChangesTotal.WithLabelValues("group", "updated"))
	assert.Equal(t, before+1, after)
}

func TestChangesTotal_AcceptsAllScopes(t *testing.T) {
	scopes := []string{"group", "user", "funnel", "view_profile"}
	for _, s := range scopes {
		before := testutil.ToFloat64(ChangesTotal.WithLabelValues(s, "updated"))
		ChangesTotal.WithLabelValues(s, "updated").Inc()
		after := testutil.ToFloat64(ChangesTotal.WithLabelValues(s, "updated"))
		assert.GreaterOrEqual(t, after, before+1, "scope %s should increment", s)
	}
}

func TestChangesTotal_Registered(t *testing.T) {
	// Touch the metric so it appears in the registry's output.
	ChangesTotal.WithLabelValues("group", "updated").Inc()

	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var names []string
	for _, m := range metrics {
		names = append(names, m.GetName())
	}
	joined := strings.Join(names, "\n")
	assert.Contains(t, joined, "crm_permission_changes_total")
}
