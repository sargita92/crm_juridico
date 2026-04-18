package infrastructure

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type GormFunnelProductRepository struct {
	db *gorm.DB
}

func NewGormFunnelProductRepository(db *gorm.DB) *GormFunnelProductRepository {
	return &GormFunnelProductRepository{db: db}
}

func (r *GormFunnelProductRepository) Create(ctx context.Context, fp *domain.FunnelProduct) error {
	model := funnelProductToModel(fp)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		if isDuplicateKeyError(err) {
			return domain.ErrFunnelProductAlreadyExists
		}
		return err
	}
	return nil
}

func (r *GormFunnelProductRepository) Delete(ctx context.Context, funnelID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("funnel_id = ? AND product_id = ?", funnelID, productID).
		Delete(&funnelProductModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrFunnelProductNotFound
	}
	return nil
}

func (r *GormFunnelProductRepository) FindByProductID(ctx context.Context, productID string) ([]domain.FunnelProduct, error) {
	var models []funnelProductModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.FunnelProduct, len(models))
	for i := range models {
		result[i] = *funnelProductToDomain(&models[i])
	}
	return result, nil
}

func (r *GormFunnelProductRepository) FindByFunnelID(ctx context.Context, funnelID string) ([]domain.FunnelProduct, error) {
	var models []funnelProductModel
	if err := r.db.WithContext(ctx).Where("funnel_id = ?", funnelID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.FunnelProduct, len(models))
	for i := range models {
		result[i] = *funnelProductToDomain(&models[i])
	}
	return result, nil
}

func (r *GormFunnelProductRepository) FindTopPriorityFunnel(ctx context.Context, tenantID, productID string) (*domain.FunnelProduct, error) {
	var model funnelProductModel
	if err := r.db.WithContext(ctx).
		Table("funnel_products").
		Joins("JOIN funnels ON funnels.id = funnel_products.funnel_id").
		Where("funnel_products.product_id = ? AND funnels.tenant_id = ?", productID, tenantID).
		Order("funnel_products.priority DESC").
		Limit(1).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFunnelProductNotFound
		}
		return nil, err
	}
	return funnelProductToDomain(&model), nil
}

func (r *GormFunnelProductRepository) UpdatePriority(ctx context.Context, funnelID, productID string, priority int) error {
	result := r.db.WithContext(ctx).
		Model(&funnelProductModel{}).
		Where("funnel_id = ? AND product_id = ?", funnelID, productID).
		Update("priority", priority)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrFunnelProductNotFound
	}
	return nil
}

// isDuplicateKeyError checks if the error is a MySQL duplicate key error.
func isDuplicateKeyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Duplicate entry")
}
