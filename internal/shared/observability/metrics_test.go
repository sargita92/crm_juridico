package observability_test

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

func TestPermissionCheckDuration_IsRegisteredWithCRMNamespace(t *testing.T) {
	observability.PermissionCheckDuration.WithLabelValues("read").Observe(0.001)

	count := testutil.CollectAndCount(observability.PermissionCheckDuration)
	assert.Greater(t, count, 0)

	out := &strings.Builder{}
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	for _, mf := range metricFamilies {
		out.WriteString(mf.GetName())
		out.WriteString("\n")
	}
	assert.Contains(t, out.String(), "crm_permission_check_duration_seconds")
}

func TestSpecialistResponseDuration_IsObservable(t *testing.T) {
	observability.SpecialistResponseDuration.WithLabelValues("ok").Observe(1.0)
	count := testutil.CollectAndCount(observability.SpecialistResponseDuration)
	assert.Greater(t, count, 0)
}
