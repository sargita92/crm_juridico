package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PermissionCheckDuration mede latência do middleware RequirePermission.
// Labels: scope (ex.: "read", "write", "admin").
// Buckets sub-ms — operação de cache-hot, caminho hot path.
var PermissionCheckDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "crm",
		Name:      "permission_check_duration_seconds",
		Help:      "Latência do middleware de checagem de permissão (segundos).",
		Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05},
	},
	[]string{"scope"},
)

// SpecialistResponseDuration mede latência da resposta do especialista IA.
// Labels: outcome ("ok", "error", "timeout").
// Buckets longos — LLM é operação de vários segundos.
var SpecialistResponseDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "crm",
		Name:      "specialist_response_duration_seconds",
		Help:      "Latência da resposta do especialista IA (segundos).",
		Buckets:   []float64{0.5, 1, 2.5, 5, 10, 20, 30, 60},
	},
	[]string{"outcome"},
)
