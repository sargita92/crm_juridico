package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type loadBalanceConfigModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null;uniqueIndex:uk_lb_tenant_group"`
	GroupID   string    `gorm:"column:group_id;type:char(36);not null;uniqueIndex:uk_lb_tenant_group"`
	Algorithm string    `gorm:"column:algorithm;type:varchar(20);not null;default:round_robin"`
	LastIndex int       `gorm:"column:last_index;not null;default:0"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (loadBalanceConfigModel) TableName() string { return "load_balance_configs" }

func loadBalanceToModel(c *domain.LoadBalanceConfig) *loadBalanceConfigModel {
	return &loadBalanceConfigModel{
		ID:        c.ID,
		TenantID:  c.TenantID,
		GroupID:   c.GroupID,
		Algorithm: string(c.Algorithm),
		LastIndex: c.LastIndex,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func loadBalanceToDomain(m *loadBalanceConfigModel) *domain.LoadBalanceConfig {
	return &domain.LoadBalanceConfig{
		ID:        m.ID,
		TenantID:  m.TenantID,
		GroupID:   m.GroupID,
		Algorithm: domain.LoadBalanceAlgorithm(m.Algorithm),
		LastIndex: m.LastIndex,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type GormLoadBalanceConfigRepository struct {
	db *gorm.DB
}

func NewGormLoadBalanceConfigRepository(db *gorm.DB) *GormLoadBalanceConfigRepository {
	return &GormLoadBalanceConfigRepository{db: db}
}

func (r *GormLoadBalanceConfigRepository) CreateOrUpdate(ctx context.Context, cfg *domain.LoadBalanceConfig) error {
	model := loadBalanceToModel(cfg)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "group_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"algorithm", "last_index", "updated_at"}),
		}).
		Create(model).Error
}

func (r *GormLoadBalanceConfigRepository) FindByGroupID(ctx context.Context, tenantID, groupID string) (*domain.LoadBalanceConfig, error) {
	var model loadBalanceConfigModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND group_id = ?", tenantID, groupID).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLoadBalanceNotFound
		}
		return nil, err
	}
	return loadBalanceToDomain(&model), nil
}

func (r *GormLoadBalanceConfigRepository) FindByTenantID(ctx context.Context, tenantID string) ([]*domain.LoadBalanceConfig, error) {
	var models []loadBalanceConfigModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]*domain.LoadBalanceConfig, len(models))
	for i := range models {
		result[i] = loadBalanceToDomain(&models[i])
	}
	return result, nil
}

func (r *GormLoadBalanceConfigRepository) Update(ctx context.Context, cfg *domain.LoadBalanceConfig) error {
	model := loadBalanceToModel(cfg)
	return r.db.WithContext(ctx).Save(model).Error
}
