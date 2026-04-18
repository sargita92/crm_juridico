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
)

func init() {
	prometheus.MustRegister(pickerTotal, pickerDuration)
}
