package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

type GormPreferenceRepository struct {
	db *gorm.DB
}

func NewGormPreferenceRepository(db *gorm.DB) *GormPreferenceRepository {
	return &GormPreferenceRepository{db: db}
}

func (r *GormPreferenceRepository) CreateOrUpdate(ctx context.Context, pref *domain.NotificationPreference) error {
	m := preferenceToModel(pref)
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *GormPreferenceRepository) FindByUser(ctx context.Context, userID, tenantID string) ([]domain.NotificationPreference, error) {
	var models []preferenceModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]domain.NotificationPreference, len(models))
	for i, m := range models {
		p := preferenceToDomain(&m)
		result[i] = *p
	}
	return result, nil
}

func (r *GormPreferenceRepository) FindByUserAndChannel(ctx context.Context, userID, tenantID string, channel domain.Channel) (*domain.NotificationPreference, error) {
	var m preferenceModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ? AND channel = ?", userID, tenantID, string(channel)).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrPreferenceNotFound
		}
		return nil, err
	}
	return preferenceToDomain(&m), nil
}

// Ensure interface is satisfied at compile time.
var _ domain.PreferenceRepository = (*GormPreferenceRepository)(nil)
