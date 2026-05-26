package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
)

// TestNew_ConnectsConfiguresPoolAndInstruments exercita o entrypoint real:
// conexão, configuração do pool e wiring de logger + tracing por query.
func TestNew_ConnectsConfiguresPoolAndInstruments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cfg := sharedContainer.Config()
	cfg.SlowQueryThresholdMs = 1 // baixo o suficiente para exercitar o caminho de slow log
	log := sharedContainer.Logger()

	db, err := database.New(cfg, log, nil)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close(db, log) })

	sqlDB, err := db.DB()
	require.NoError(t, err)
	assert.Equal(t, 25, sqlDB.Stats().MaxOpenConnections, "pool deve ser configurado em New")

	// Query real passa pelos callbacks de tracing e pelo logger sem erro.
	var n int
	require.NoError(t, db.Raw("SELECT 1").Scan(&n).Error)
	assert.Equal(t, 1, n)
}
