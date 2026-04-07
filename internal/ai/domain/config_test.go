package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAIConfig_WithSpecialist_ReturnsConfig(t *testing.T) {
	cfg, err := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	require.NoError(t, err)
	assert.Equal(t, "id-1", cfg.ID)
	assert.Equal(t, "spec-1", cfg.SpecialistID)
	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "gpt-4", cfg.Model)
	assert.Equal(t, 0.7, cfg.Temperature)
	assert.Equal(t, 1000, cfg.MaxTokens)
	assert.Equal(t, 30, cfg.DebounceSeconds)
	assert.False(t, cfg.CreatedAt.IsZero())
	assert.False(t, cfg.UpdatedAt.IsZero())
}

func TestNewAIConfig_GlobalConfig_EmptySpecialistID_ReturnsConfig(t *testing.T) {
	cfg, err := NewAIConfig("id-1", "", "openai", "gpt-4", 0.5, 500, 0)

	require.NoError(t, err)
	assert.Empty(t, cfg.SpecialistID)
}

func TestNewAIConfig_EmptyProvider_ReturnsError(t *testing.T) {
	_, err := NewAIConfig("id-1", "spec-1", "", "gpt-4", 0.7, 1000, 30)

	assert.ErrorIs(t, err, ErrAIProviderRequired)
}

func TestNewAIConfig_EmptyModel_ReturnsError(t *testing.T) {
	_, err := NewAIConfig("id-1", "spec-1", "openai", "", 0.7, 1000, 30)

	assert.ErrorIs(t, err, ErrModelRequired)
}

func TestNewAIConfig_TemperatureAboveOne_ReturnsError(t *testing.T) {
	_, err := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 1.5, 1000, 30)

	assert.ErrorIs(t, err, ErrTemperatureInvalid)
}

func TestNewAIConfig_TemperatureBelowZero_ReturnsError(t *testing.T) {
	_, err := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", -0.1, 1000, 30)

	assert.ErrorIs(t, err, ErrTemperatureInvalid)
}

func TestNewAIConfig_NegativeDebounce_ReturnsError(t *testing.T) {
	_, err := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, -1)

	assert.ErrorIs(t, err, ErrDebounceInvalid)
}

func TestAIConfig_Update_Success(t *testing.T) {
	cfg, _ := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	err := cfg.Update("anthropic", "claude-3", 0.5, 2000, 60)

	require.NoError(t, err)
	assert.Equal(t, "anthropic", cfg.Provider)
	assert.Equal(t, "claude-3", cfg.Model)
	assert.Equal(t, 0.5, cfg.Temperature)
	assert.Equal(t, 2000, cfg.MaxTokens)
	assert.Equal(t, 60, cfg.DebounceSeconds)
}

func TestAIConfig_Update_EmptyProvider_ReturnsError(t *testing.T) {
	cfg, _ := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	err := cfg.Update("", "gpt-4", 0.7, 1000, 30)

	assert.ErrorIs(t, err, ErrAIProviderRequired)
	assert.Equal(t, "openai", cfg.Provider)
}

func TestAIConfig_Update_EmptyModel_ReturnsError(t *testing.T) {
	cfg, _ := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	err := cfg.Update("openai", "", 0.7, 1000, 30)

	assert.ErrorIs(t, err, ErrModelRequired)
	assert.Equal(t, "gpt-4", cfg.Model)
}

func TestAIConfig_Update_InvalidTemperature_ReturnsError(t *testing.T) {
	cfg, _ := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	err := cfg.Update("openai", "gpt-4", 2.0, 1000, 30)

	assert.ErrorIs(t, err, ErrTemperatureInvalid)
	assert.Equal(t, 0.7, cfg.Temperature)
}

func TestAIConfig_Update_NegativeDebounce_ReturnsError(t *testing.T) {
	cfg, _ := NewAIConfig("id-1", "spec-1", "openai", "gpt-4", 0.7, 1000, 30)

	err := cfg.Update("openai", "gpt-4", 0.7, 1000, -5)

	assert.ErrorIs(t, err, ErrDebounceInvalid)
	assert.Equal(t, 30, cfg.DebounceSeconds)
}
