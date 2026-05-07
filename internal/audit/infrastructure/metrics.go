package infrastructure

import "github.com/prometheus/client_golang/prometheus"

// Metrics do dominio audit. Registradas globalmente via init() — padrao
// vigente no projeto (ver internal/auth/infrastructure/metrics.go).
//
// Naming/labels seguem secao 9.1 do design F12:
//   - crm_audit_logs_registered_total{action,status}
//   - crm_audit_logs_list_duration_seconds (sem labels)
var (
	// AuditLogsRegisteredTotal conta tentativas de gravar audit logs por
	// action e status (success|error). Incrementado pelo
	// RegisterAuditLogUseCase em ambos os caminhos.
	AuditLogsRegisteredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "audit",
			Name:      "logs_registered_total",
			Help:      "Total de audit logs publicados, por action e status (success|error).",
		},
		[]string{"action", "status"},
	)

	// AuditLogsListDuration mede a latencia da listagem de audit logs
	// (filtro + paginacao). Usado pelo ListAuditLogsUseCase no Step 4.
	// Exposto agora para consolidar o registro de metrics em um unico lugar.
	AuditLogsListDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "crm",
			Subsystem: "audit",
			Name:      "logs_list_duration_seconds",
			Help:      "Latencia da listagem de audit logs (filtro + paginacao).",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{},
	)
)

func init() {
	prometheus.MustRegister(AuditLogsRegisteredTotal, AuditLogsListDuration)
}
