package observability

import (
	"strings"
	"testing"

	autinfra "github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	perminfra "github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestAllF08F09MetricsRegistered(t *testing.T) {
	// Touch each metric so the counter vec appears in the gatherer output.
	autinfra.ExecutionsTotal.WithLabelValues("auto_message", "success").Add(0)
	perminfra.ChangesTotal.WithLabelValues("group", "updated").Add(0)
	authinfra.InvitesTotal.WithLabelValues("sent").Add(0)

	required := []string{
		"crm_automation_executions_total",
		"crm_permission_changes_total",
		"crm_auth_invites_total",
	}
	mfs, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)
	found := make(map[string]bool)
	for _, mf := range mfs {
		found[mf.GetName()] = true
	}
	for _, name := range required {
		assert.True(t, found[name], "expected metric %s to be registered; have: %s",
			name, strings.Join(keys(found), ", "))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
