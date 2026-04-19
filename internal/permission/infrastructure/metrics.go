package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// ChangesTotal counts permission-scope changes, broken down by scope and action.
// scope ∈ {group, user, funnel, view_profile}
// action ∈ {updated}
var ChangesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "permission",
		Name:      "changes_total",
		Help:      "Total permission-scope changes, by scope and action.",
	},
	[]string{"scope", "action"},
)

func init() {
	prometheus.MustRegister(ChangesTotal)
}
