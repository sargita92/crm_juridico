package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// LoadBalanceFallbackTotal counts how often LoadBalancePicker.PickForFunnel
// had to fall back to the tenant owner instead of picking a user via a load
// balance group.
//
// Label reason mirrors the internal codes used by load_balance_picker.go:
//   - "no_group"               — no group covers the column
//   - "no_active_config"       — covering groups exist but none is active
//   - "multiple_active_groups" — more than one active group (uniqueness violation)
//   - "no_active_members"      — active group has no tenant-active members
//   - "group_lookup_error"     — infra error in groupFunnelRepo
//   - "lb_lookup_error"        — infra error in lbRepo
//   - "member_lookup_error"    — infra error in userGroupRepo
//   - "member_check_error"     — infra error in userTenantRepo
//   - "algorithm_error"        — error applying the algorithm
var LoadBalanceFallbackTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "load_balance",
		Name:      "fallback_total",
		Help:      "Total responsible-picker fallbacks to the tenant owner, by reason.",
	},
	[]string{"reason"},
)

func init() {
	prometheus.MustRegister(LoadBalanceFallbackTotal)
}
