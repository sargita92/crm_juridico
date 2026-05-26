package observability_test

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql" // registra o driver "mysql" para sql.Open (conexão lazy, sem rede)
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// TestRegisterDBStats_ExposesPoolMetrics garante que as métricas do pool de
// conexões (sql.DBStats) ficam disponíveis no registry Prometheus — em
// particular wait_count/wait_duration, que são a evidência direta de exaustão
// de pool investigada na F26.
func TestRegisterDBStats_ExposesPoolMetrics(t *testing.T) {
	// sql.Open é lazy: não abre conexão de rede, mas db.Stats() já funciona.
	db, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:3306)/crm_test")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	reg := prometheus.NewRegistry()
	require.NoError(t, observability.RegisterDBStats(reg, db, "crm_test"))

	mfs, err := reg.Gather()
	require.NoError(t, err)

	found := make(map[string]bool, len(mfs))
	for _, mf := range mfs {
		found[mf.GetName()] = true
	}

	required := []string{
		"go_sql_open_connections",
		"go_sql_in_use_connections",
		"go_sql_idle_connections",
		"go_sql_wait_count_total",
		"go_sql_wait_duration_seconds_total",
	}
	for _, name := range required {
		assert.Truef(t, found[name], "métrica %q deveria estar registrada; tenho: %s",
			name, strings.Join(keysOf(found), ", "))
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
