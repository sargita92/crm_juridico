package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/shared/config"
)

// TestLoad_Defaults garante que Load aplica os defaults esperados, incluindo os
// introduzidos na F26 (slow-query threshold e pprof desabilitado).
func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)

	// F26
	assert.Equal(t, 200, cfg.Database.SlowQueryThresholdMs, "slow-query threshold default = 200ms")
	assert.False(t, cfg.Server.PprofEnabled, "pprof deve vir desabilitado por padrão")
	assert.Equal(t, 1000, cfg.Server.SlowRequestThresholdMs, "slow-request threshold default = 1000ms")

	// Sanity de alguns defaults pré-existentes.
	assert.Equal(t, "8533", cfg.Server.Port)
	assert.Equal(t, "crm_juridico", cfg.Database.Name)
}

// TestDSN monta a string de conexão a partir do DatabaseConfig.
func TestDSN(t *testing.T) {
	d := config.DatabaseConfig{
		Host: "db", Port: "3306", User: "u", Password: "p", Name: "crm",
	}
	assert.Equal(t,
		"u:p@tcp(db:3306)/crm?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		d.DSN(),
	)
}
