package infrastructure_test

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// TestLoadBalanceFallbackTotal_IncrementsOnFallback drives a picker through a
// known fallback reason ("no_group") and verifies the counter series for that
// reason went up by one.
func TestLoadBalanceFallbackTotal_IncrementsOnFallback(t *testing.T) {
	picker := infrastructure.NewLoadBalancePicker(
		&fakeGroupFunnelRepo{byFunnel: map[string][]permdomain.GroupFunnel{}},
		&fakeLoadBalanceRepo{byGroup: map[string]*authdomain.LoadBalanceConfig{}},
		&fakeUserGroupRepo{byGroup: map[string][]permdomain.UserGroup{}},
		&fakeUserTenantRepo{ownerByTenant: map[string]string{"t1": "owner-1"}},
		&fakeLoadCounter{},
		zap.NewNop(),
	)

	before := testutil.ToFloat64(infrastructure.LoadBalanceFallbackTotal.WithLabelValues("no_group"))

	_, err := picker.PickForFunnel(context.Background(), "t1", "f1", "c1")
	require.NoError(t, err)

	after := testutil.ToFloat64(infrastructure.LoadBalanceFallbackTotal.WithLabelValues("no_group"))
	assert.Equal(t, before+1, after, "fallback counter for reason=no_group should increment once")
}

func TestLoadBalanceFallbackTotal_Registered(t *testing.T) {
	// Touch the counter at zero so the family appears in the registry.
	infrastructure.LoadBalanceFallbackTotal.WithLabelValues("no_group").Add(0)

	mfs, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var names []string
	for _, m := range mfs {
		names = append(names, m.GetName())
	}
	joined := strings.Join(names, "\n")
	assert.Contains(t, joined, "crm_load_balance_fallback_total")
}
