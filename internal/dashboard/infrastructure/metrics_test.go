package infrastructure_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/dashboard/infrastructure"
)

func TestMetricsRegistered(t *testing.T) {
	// RenderDuration and LoadTotal exist (compilation guarantees this) and
	// are registered via init() in the same package.
	infrastructure.RenderDuration.WithLabelValues("tenant").Observe(0.1)
	infrastructure.LoadTotal.WithLabelValues("tenant", "success").Inc()

	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)

	var foundDur, foundLoad bool
	for _, mf := range mfs {
		if strings.Contains(mf.GetName(), "dashboard_render_duration_seconds") {
			foundDur = true
		}
		if strings.Contains(mf.GetName(), "dashboard_load_total") {
			foundLoad = true
		}
	}
	require.True(t, foundDur, "dashboard_render_duration_seconds não registrada")
	require.True(t, foundLoad, "dashboard_load_total não registrada")
}
