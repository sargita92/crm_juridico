package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// PreferenceOutput is the read model for notification preferences.
type PreferenceOutput struct {
	ID        string
	UserID    string
	TenantID  string
	Channel   domain.Channel
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ManagePreferencesUseCase handles reading and writing notification preferences.
type ManagePreferencesUseCase struct {
	repo domain.PreferenceRepository
}

func NewManagePreferencesUseCase(repo domain.PreferenceRepository) *ManagePreferencesUseCase {
	return &ManagePreferencesUseCase{repo: repo}
}

// GetPreferences returns all preferences for a user in a tenant.
func (uc *ManagePreferencesUseCase) GetPreferences(ctx context.Context, userID, tenantID string) ([]PreferenceOutput, error) {
	items, err := uc.repo.FindByUser(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	outputs := make([]PreferenceOutput, len(items))
	for i, p := range items {
		outputs[i] = toPreferenceOutput(p)
	}
	return outputs, nil
}

// SetPreference creates or updates a notification preference for a specific channel.
func (uc *ManagePreferencesUseCase) SetPreference(ctx context.Context, userID, tenantID string, channel domain.Channel, enabled bool) error {
	// Try to find existing preference.
	existing, err := uc.repo.FindByUserAndChannel(ctx, userID, tenantID, channel)
	if err != nil && err != domain.ErrPreferenceNotFound {
		return err
	}

	var pref *domain.NotificationPreference
	if existing != nil {
		existing.Enabled = enabled
		existing.UpdatedAt = time.Now()
		pref = existing
	} else {
		pref, err = domain.NewNotificationPreference(uuid.New().String(), userID, tenantID, channel, enabled)
		if err != nil {
			return err
		}
	}

	return uc.repo.CreateOrUpdate(ctx, pref)
}

func toPreferenceOutput(p domain.NotificationPreference) PreferenceOutput {
	return PreferenceOutput{
		ID:        p.ID,
		UserID:    p.UserID,
		TenantID:  p.TenantID,
		Channel:   p.Channel,
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
