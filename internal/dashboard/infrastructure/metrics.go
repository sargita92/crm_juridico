package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// Métricas do dashboard F19.
// scope: "tenant" | "admin"
// outcome: "success" | "error"
var (
	RenderDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dashboard_render_duration_seconds",
			Help:    "Tempo total de renderização do dashboard (do handler até o HTML final).",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"scope"},
	)
	LoadTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dashboard_load_total",
			Help: "Total de carregamentos do dashboard, por escopo e desfecho.",
		},
		[]string{"scope", "outcome"},
	)
)

func init() {
	prometheus.MustRegister(RenderDuration, LoadTotal)
}
