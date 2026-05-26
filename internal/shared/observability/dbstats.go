package observability

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// RegisterDBStats expõe as estatísticas do pool de conexões (sql.DBStats) no
// registry Prometheus informado. As métricas resultantes têm prefixo go_sql_ e
// incluem open/in_use/idle_connections, max_open_connections e — o que mais
// importa para diagnosticar gargalos — wait_count_total e
// wait_duration_seconds_total, que crescem quando o pool está saturado e os
// requests passam a aguardar uma conexão livre.
//
// dbName vira o label db_name, permitindo distinguir múltiplos pools no futuro.
func RegisterDBStats(reg prometheus.Registerer, db *sql.DB, dbName string) error {
	return reg.Register(collectors.NewDBStatsCollector(db, dbName))
}
