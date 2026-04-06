package infrastructure

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type GormTenantProductRepository struct {
	db *gorm.DB
}

func NewGormTenantProductRepository(db *gorm.DB) *GormTenantProductRepository {
	return &GormTenantProductRepository{db: db}
}

func (r *GormTenantProductRepository) Create(ctx context.Context, tp *domain.TenantProduct) error {
	model := tenantProductToModel(tp)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if isTenantProductDuplicateKeyError(err) {
			return domain.ErrTenantProductAlreadyExists
		}
		return err
	}
	return nil
}

func (r *GormTenantProductRepository) Delete(ctx context.Context, tenantID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Delete(&tenantProductModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrTenantProductNotFound
	}
	return nil
}

func (r *GormTenantProductRepository) FindByTenantID(ctx context.Context, tenantID string) ([]domain.TenantProduct, error) {
	var models []tenantProductModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.TenantProduct, len(models))
	for i := range models {
		result[i] = *tenantProductToDomain(&models[i])
	}
	return result, nil
}

func (r *GormTenantProductRepository) FindByProductID(ctx context.Context, productID string) ([]domain.TenantProduct, error) {
	var models []tenantProductModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.TenantProduct, len(models))
	for i := range models {
		result[i] = *tenantProductToDomain(&models[i])
	}
	return result, nil
}

func (r *GormTenantProductRepository) Exists(ctx context.Context, tenantID, productID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&tenantProductModel{}).
		Where("tenant_id = ? AND product_id = ?", tenantID, productID).
		Count(&count).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}

func isTenantProductDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}
