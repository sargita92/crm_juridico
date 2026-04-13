package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoadBalanceConfig_RoundRobin(t *testing.T) {
	cfg, err := NewLoadBalanceConfig("id-1", "tenant-1", "group-1", AlgorithmRoundRobin)
	require.NoError(t, err)
	assert.Equal(t, "id-1", cfg.ID)
	assert.Equal(t, AlgorithmRoundRobin, cfg.Algorithm)
	assert.Equal(t, 0, cfg.LastIndex)
}

func TestNewLoadBalanceConfig_AllAlgorithms(t *testing.T) {
	valid := []LoadBalanceAlgorithm{AlgorithmRoundRobin, AlgorithmLeastLoad, AlgorithmRandom}
	for _, a := range valid {
		t.Run(string(a), func(t *testing.T) {
			_, err := NewLoadBalanceConfig("id", "tenant", "group", a)
			assert.NoError(t, err)
		})
	}
}

func TestNewLoadBalanceConfig_EmptyTenantID(t *testing.T) {
	_, err := NewLoadBalanceConfig("id", "", "group", AlgorithmRoundRobin)
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewLoadBalanceConfig_EmptyGroupID(t *testing.T) {
	_, err := NewLoadBalanceConfig("id", "tenant", "", AlgorithmRoundRobin)
	assert.ErrorIs(t, err, ErrGroupIDRequired)
}

func TestNewLoadBalanceConfig_InvalidAlgorithm(t *testing.T) {
	_, err := NewLoadBalanceConfig("id", "tenant", "group", "weighted_nonsense")
	assert.ErrorIs(t, err, ErrInvalidAlgorithm)
}

func TestLoadBalanceConfig_IncrementIndex(t *testing.T) {
	cfg, err := NewLoadBalanceConfig("id", "tenant", "group", AlgorithmRoundRobin)
	require.NoError(t, err)

	assert.Equal(t, 1, cfg.IncrementIndex())
	assert.Equal(t, 2, cfg.IncrementIndex())
	assert.Equal(t, 3, cfg.IncrementIndex())
	assert.Equal(t, 3, cfg.LastIndex)
}
