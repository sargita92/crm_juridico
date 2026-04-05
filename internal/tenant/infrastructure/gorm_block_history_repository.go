package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type blockHistoryModel struct {
	ID          string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID    string    `gorm:"column:tenant_id;type:char(36);not null"`
	Action      string    `gorm:"column:action;type:enum('block','unblock');not null"`
	Reason      string    `gorm:"column:reason;type:varchar(500);not null"`
	PerformedBy string    `gorm:"column:performed_by;type:char(36);not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (blockHistoryModel) TableName() string { return "tenant_block_history" }

func toHistoryModel(e *domain.BlockHistoryEntry) *blockHistoryModel {
	return &blockHistoryModel{
		ID:          e.ID,
		TenantID:    e.TenantID,
		Action:      string(e.Action),
		Reason:      e.Reason,
		PerformedBy: e.PerformedBy,
		CreatedAt:   e.CreatedAt,
	}
}

func toHistoryDomain(m *blockHistoryModel) domain.BlockHistoryEntry {
	return domain.BlockHistoryEntry{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Action:      domain.BlockAction(m.Action),
		Reason:      m.Reason,
		PerformedBy: m.PerformedBy,
		CreatedAt:   m.CreatedAt,
	}
}

type GormBlockHistoryRepository struct {
	db *gorm.DB
}

func NewGormBlockHistoryRepository(db *gorm.DB) *GormBlockHistoryRepository {
	return &GormBlockHistoryRepository{db: db}
}

func (r *GormBlockHistoryRepository) Save(ctx context.Context, entry *domain.BlockHistoryEntry) error {
	model := toHistoryModel(entry)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormBlockHistoryRepository) FindByTenantID(ctx context.Context, tenantID string) ([]domain.BlockHistoryEntry, error) {
	var models []blockHistoryModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	entries := make([]domain.BlockHistoryEntry, len(models))
	for i := range models {
		entries[i] = toHistoryDomain(&models[i])
	}
	return entries, nil
}
