package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type specialistTenantModel struct {
	SpecialistID string    `gorm:"primaryKey;column:specialist_id;type:char(36)"`
	TenantID     string    `gorm:"primaryKey;column:tenant_id;type:char(36)"`
	IsDefault    bool      `gorm:"column:is_default;not null;default:false"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistTenantModel) TableName() string { return "specialist_tenants" }

type GormSpecialistTenantRepository struct {
	db *gorm.DB
}

func NewGormSpecialistTenantRepository(db *gorm.DB) *GormSpecialistTenantRepository {
	return &GormSpecialistTenantRepository{db: db}
}

func (r *GormSpecialistTenantRepository) Associate(ctx context.Context, specialistID, tenantID string) error {
	model := specialistTenantModel{
		SpecialistID: specialistID,
		TenantID:     tenantID,
		CreatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrTenantAlreadyAssociated
		}
		return err
	}
	return nil
}

func (r *GormSpecialistTenantRepository) Dissociate(ctx context.Context, specialistID, tenantID string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND tenant_id = ?", specialistID, tenantID).
		Delete(&specialistTenantModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrTenantNotAssociated
	}
	return nil
}

func (r *GormSpecialistTenantRepository) FindTenantIDsBySpecialistID(ctx context.Context, specialistID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&specialistTenantModel{}).
		Where("specialist_id = ?", specialistID).
		Pluck("tenant_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistTenantRepository) FindSpecialistIDsByTenantID(ctx context.Context, tenantID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&specialistTenantModel{}).
		Where("tenant_id = ?", tenantID).
		Pluck("specialist_id", &ids).Error
	return ids, err
}

func (r *GormSpecialistTenantRepository) Exists(ctx context.Context, specialistID, tenantID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&specialistTenantModel{}).
		Where("specialist_id = ? AND tenant_id = ?", specialistID, tenantID).
		Count(&count).Error
	return count > 0, err
}

func (r *GormSpecialistTenantRepository) FindDefaultByTenantID(ctx context.Context, tenantID string) (string, error) {
	var model specialistTenantModel
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND is_default = ?", tenantID, true).
		First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", domain.ErrSpecialistNotFound
		}
		return "", err
	}
	return model.SpecialistID, nil
}

func (r *GormSpecialistTenantRepository) SetDefault(ctx context.Context, specialistID, tenantID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&specialistTenantModel{}).
			Where("tenant_id = ? AND is_default = ?", tenantID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		result := tx.Model(&specialistTenantModel{}).
			Where("specialist_id = ? AND tenant_id = ?", specialistID, tenantID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrTenantNotAssociated
		}
		return nil
	})
}
