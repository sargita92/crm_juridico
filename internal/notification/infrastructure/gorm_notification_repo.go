package infrastructure

import (
	"context"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

type GormNotificationRepository struct {
	db *gorm.DB
}

func NewGormNotificationRepository(db *gorm.DB) *GormNotificationRepository {
	return &GormNotificationRepository{db: db}
}

func (r *GormNotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	m := notificationToModel(n)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *GormNotificationRepository) FindByUserID(ctx context.Context, tenantID, userID string, onlyUnread bool, limit, offset int) ([]domain.Notification, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("created_at DESC")

	if onlyUnread {
		query = query.Where("is_read = ?", false)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var models []notificationModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]domain.Notification, len(models))
	for i, m := range models {
		n := notificationToDomain(&m)
		result[i] = *n
	}
	return result, nil
}

func (r *GormNotificationRepository) CountUnread(ctx context.Context, tenantID, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&notificationModel{}).
		Where("tenant_id = ? AND user_id = ? AND is_read = ?", tenantID, userID, false).
		Count(&count).Error
	return count, err
}

func (r *GormNotificationRepository) MarkRead(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Model(&notificationModel{}).
		Where("id = ?", id).
		Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *GormNotificationRepository) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	return r.db.WithContext(ctx).
		Model(&notificationModel{}).
		Where("tenant_id = ? AND user_id = ? AND is_read = ?", tenantID, userID, false).
		Update("is_read", true).Error
}

// Ensure interface is satisfied at compile time.
var _ domain.NotificationRepository = (*GormNotificationRepository)(nil)
