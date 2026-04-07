package application

import (
	"context"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/config"
)

// EnvConfigResolver resolves AI configuration from the database with a fallback to environment defaults.
type EnvConfigResolver struct {
	repo   domain.AIConfigRepository
	envCfg config.AIConfigEnv
}

// NewEnvConfigResolver creates an EnvConfigResolver with a repository and env-based defaults.
func NewEnvConfigResolver(repo domain.AIConfigRepository, envCfg config.AIConfigEnv) *EnvConfigResolver {
	return &EnvConfigResolver{repo: repo, envCfg: envCfg}
}

// Resolve returns the AIConfig for the given specialistID.
// It first tries the repository; if not found or on error, it returns a config built from env defaults.
func (r *EnvConfigResolver) Resolve(ctx context.Context, specialistID string) *domain.AIConfig {
	if specialistID != "" {
		cfg, err := r.repo.FindBySpecialistID(ctx, specialistID)
		if err == nil && cfg != nil {
			return cfg
		}
	}

	return &domain.AIConfig{
		SpecialistID: specialistID,
		Provider:     r.envCfg.DefaultProvider,
		Model:        r.envCfg.DefaultModel,
		Temperature:  r.envCfg.DefaultTemperature,
		MaxTokens:    r.envCfg.DefaultMaxTokens,
		DebounceSeconds: r.envCfg.DefaultDebounce,
	}
}
