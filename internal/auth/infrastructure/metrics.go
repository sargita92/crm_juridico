package infrastructure

import "github.com/prometheus/client_golang/prometheus"

var (
	pickerTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "lead_responsible_picker",
			Name:      "total",
			Help:      "Total responsible-picker attempts broken down by algorithm and outcome.",
		},
		[]string{"algorithm", "outcome"},
	)

	pickerDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "crm",
			Subsystem: "lead_responsible_picker",
			Name:      "duration_seconds",
			Help:      "Duration of responsible-picker calls in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"algorithm"},
	)

	// InvitesTotal counts invite token lifecycle events, by outcome.
	// outcome ∈ {sent, accepted, revoked}
	InvitesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "auth",
			Name:      "invites_total",
			Help:      "Total invite token events, by outcome.",
		},
		[]string{"outcome"},
	)
)

func init() {
	prometheus.MustRegister(pickerTotal, pickerDuration, InvitesTotal)
}
