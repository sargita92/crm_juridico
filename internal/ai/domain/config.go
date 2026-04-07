package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrAIProviderRequired = errors.New("AI provider is required")
	ErrDebounceInvalid    = errors.New("debounce seconds must be >= 0")
	ErrAIConfigNotFound   = errors.New("AI config not found")
)

// AIConfig holds the AI configuration for a specialist or globally.
type AIConfig struct {
	ID              string
	SpecialistID    string // empty = global config
	Provider        string
	Model           string
	Temperature     float64
	MaxTokens       int
	DebounceSeconds int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewAIConfig constructs an AIConfig with validation.
func NewAIConfig(id, specialistID, provider, model string, temperature float64, maxTokens, debounceSeconds int) (*AIConfig, error) {
	if err := validateAIConfigFields(provider, model, temperature, debounceSeconds); err != nil {
		return nil, err
	}
	now := time.Now()
	return &AIConfig{
		ID:              id,
		SpecialistID:    specialistID,
		Provider:        provider,
		Model:           model,
		Temperature:     temperature,
		MaxTokens:       maxTokens,
		DebounceSeconds: debounceSeconds,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// Update changes the configuration fields with validation.
func (c *AIConfig) Update(provider, model string, temperature float64, maxTokens, debounceSeconds int) error {
	if err := validateAIConfigFields(provider, model, temperature, debounceSeconds); err != nil {
		return err
	}
	c.Provider = provider
	c.Model = model
	c.Temperature = temperature
	c.MaxTokens = maxTokens
	c.DebounceSeconds = debounceSeconds
	c.UpdatedAt = time.Now()
	return nil
}

func validateAIConfigFields(provider, model string, temperature float64, debounceSeconds int) error {
	if provider == "" {
		return ErrAIProviderRequired
	}
	if model == "" {
		return ErrModelRequired
	}
	if temperature < 0 || temperature > 1 {
		return ErrTemperatureInvalid
	}
	if debounceSeconds < 0 {
		return ErrDebounceInvalid
	}
	return nil
}

// AIConfigRepository defines persistence operations for AIConfig.
type AIConfigRepository interface {
	CreateOrUpdate(ctx context.Context, config *AIConfig) error
	FindBySpecialistID(ctx context.Context, specialistID string) (*AIConfig, error)
	FindGlobal(ctx context.Context) (*AIConfig, error)
}
