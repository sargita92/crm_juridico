package infrastructure

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CronRunsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pagamentos_cron_runs_total",
		Help: "Total de execucoes do cron de pagamentos por status.",
	}, []string{"status"})

	CronDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pagamentos_cron_duration_seconds",
		Help:    "Duracao das execucoes do cron de pagamentos.",
		Buckets: prometheus.DefBuckets,
	})

	RecorrentesGeradosTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pagamentos_recorrentes_gerados_total",
		Help: "Total de lancamentos recorrentes criados pelo cron.",
	})

	AtualizadosAtrasadoTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pagamentos_atualizados_atrasado_total",
		Help: "Lancamentos transitados para status atrasado.",
	})

	MarcadosPagoTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pagamentos_marcados_pago_total",
		Help: "Lancamentos marcados como pagos via UI admin.",
	})

	LancadosAvulsoTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pagamentos_lancados_avulso_total",
		Help: "Lancamentos avulsos criados via UI admin.",
	})

	CanceladosTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pagamentos_cancelados_total",
		Help: "Lancamentos cancelados via UI admin.",
	})
)
