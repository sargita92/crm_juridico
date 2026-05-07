package application

import (
	"context"
	"testing"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock AIConfigRepository ---

type mockAIConfigRepo struct {
	cfg     *domain.AIConfig
	findErr error
}

func (m *mockAIConfigRepo) CreateOrUpdate(_ context.Context, _ *domain.AIConfig) error {
	return nil
}

func (m *mockAIConfigRepo) FindBySpecialistID(_ context.Context, _ string) (*domain.AIConfig, error) {
	return m.cfg, m.findErr
}

func (m *mockAIConfigRepo) FindGlobal(_ context.Context) (*domain.AIConfig, error) {
	return m.cfg, m.findErr
}

// --- tests ---

func TestEnvConfigResolver_UsesSpecialistConfig(t *testing.T) {
	specialistCfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "anthropic", "claude-3", 0.5, 2048, 5)

	repo := &mockAIConfigRepo{cfg: specialistCfg}
	envCfg := config.AIConfigEnv{
		DefaultProvider:    "openai",
		DefaultModel:       "gpt-4",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   1024,
		DefaultDebounce:    8,
	}

	resolver := NewEnvConfigResolver(repo, envCfg)
	result := resolver.Resolve(context.Background(), "spec-1")

	require.NotNil(t, result)
	assert.Equal(t, "anthropic", result.Provider)
	assert.Equal(t, "claude-3", result.Model)
	assert.Equal(t, 0.5, result.Temperature)
	assert.Equal(t, 2048, result.MaxTokens)
}

func TestEnvConfigResolver_FallsBackToEnv(t *testing.T) {
	repo := &mockAIConfigRepo{cfg: nil, findErr: domain.ErrAIConfigNotFound}
	envCfg := config.AIConfigEnv{
		DefaultProvider:    "openai",
		DefaultModel:       "gpt-4",
		DefaultTemperature: 0.7,
		DefaultMaxTokens:   1024,
		DefaultDebounce:    8,
	}

	resolver := NewEnvConfigResolver(repo, envCfg)
	result := resolver.Resolve(context.Background(), "spec-unknown")

	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, "gpt-4", result.Model)
	assert.Equal(t, 0.7, result.Temperature)
	assert.Equal(t, 1024, result.MaxTokens)
	assert.Equal(t, 8, result.DebounceSeconds)
}

func TestEnvConfigResolver_FallsBackToEnv_WhenSpecialistIDEmpty(t *testing.T) {
	repo := &mockAIConfigRepo{}
	envCfg := config.AIConfigEnv{
		DefaultProvider:    "openai",
		DefaultModel:       "gpt-5-mini",
		DefaultTemperature: 0.3,
		DefaultMaxTokens:   512,
		DefaultDebounce:    3,
	}

	resolver := NewEnvConfigResolver(repo, envCfg)
	result := resolver.Resolve(context.Background(), "")

	require.NotNil(t, result)
	assert.Equal(t, "openai", result.Provider)
	assert.Equal(t, "gpt-5-mini", result.Model)
}
