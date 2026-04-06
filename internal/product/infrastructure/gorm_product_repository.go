package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type GormProductRepository struct {
	db *gorm.DB
}

func NewGormProductRepository(db *gorm.DB) *GormProductRepository {
	return &GormProductRepository{db: db}
}

func (r *GormProductRepository) Create(ctx context.Context, p *domain.Product) error {
	model := productToModel(p)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	var model productModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return productToDomain(&model), nil
}

func (r *GormProductRepository) Update(ctx context.Context, p *domain.Product) error {
	model := productToModel(p)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *GormProductRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&productModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *GormProductRepository) FindAll(ctx context.Context, activeOnly bool) ([]domain.Product, error) {
	query := r.db.WithContext(ctx)
	if activeOnly {
		query = query.Where("active = ?", true)
	}
	var models []productModel
	if err := query.Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	products := make([]domain.Product, len(models))
	for i := range models {
		products[i] = *productToDomain(&models[i])
	}
	return products, nil
}

func (r *GormProductRepository) FindActiveByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	if len(ids) == 0 {
		return []domain.Product{}, nil
	}
	var models []productModel
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND active = ?", ids, true).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	products := make([]domain.Product, len(models))
	for i := range models {
		products[i] = *productToDomain(&models[i])
	}
	return products, nil
}
