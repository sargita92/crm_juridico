package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type specialistProductModel struct {
	ID           string `gorm:"primaryKey;column:id;type:char(36)"`
	SpecialistID string `gorm:"column:specialist_id;type:char(36);not null;index"`
	ProductID    string `gorm:"column:product_id;type:char(36);not null;index"`
	CreatedAt    time.Time
}

func (specialistProductModel) TableName() string { return "specialist_products" }

func specialistProductToModel(sp *domain.SpecialistProduct) *specialistProductModel {
	return &specialistProductModel{
		ID:           sp.ID,
		SpecialistID: sp.SpecialistID,
		ProductID:    sp.ProductID,
		CreatedAt:    sp.CreatedAt,
	}
}

func specialistProductToDomain(m *specialistProductModel) *domain.SpecialistProduct {
	return &domain.SpecialistProduct{
		ID:           m.ID,
		SpecialistID: m.SpecialistID,
		ProductID:    m.ProductID,
		CreatedAt:    m.CreatedAt,
	}
}

// GormSpecialistProductRepository implements domain.SpecialistProductRepository using GORM.
type GormSpecialistProductRepository struct {
	db *gorm.DB
}

// NewGormSpecialistProductRepository creates a new GormSpecialistProductRepository.
func NewGormSpecialistProductRepository(db *gorm.DB) *GormSpecialistProductRepository {
	return &GormSpecialistProductRepository{db: db}
}

// Create inserts a new SpecialistProduct record.
func (r *GormSpecialistProductRepository) Create(ctx context.Context, sp *domain.SpecialistProduct) error {
	model := specialistProductToModel(sp)
	return r.db.WithContext(ctx).Create(model).Error
}

// Delete removes the link between a specialist and a product.
func (r *GormSpecialistProductRepository) Delete(ctx context.Context, specialistID, productID string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND product_id = ?", specialistID, productID).
		Delete(&specialistProductModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrSpecialistProductNotFound
	}
	return nil
}

// FindBySpecialistID retrieves all products linked to a specialist.
func (r *GormSpecialistProductRepository) FindBySpecialistID(ctx context.Context, specialistID string) ([]domain.SpecialistProduct, error) {
	var models []specialistProductModel
	if err := r.db.WithContext(ctx).Where("specialist_id = ?", specialistID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SpecialistProduct, len(models))
	for i := range models {
		result[i] = *specialistProductToDomain(&models[i])
	}
	return result, nil
}

// FindByProductID retrieves all specialists linked to a product.
func (r *GormSpecialistProductRepository) FindByProductID(ctx context.Context, productID string) ([]domain.SpecialistProduct, error) {
	var models []specialistProductModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.SpecialistProduct, len(models))
	for i := range models {
		result[i] = *specialistProductToDomain(&models[i])
	}
	return result, nil
}

// Exists checks whether a specialist-product link exists.
func (r *GormSpecialistProductRepository) Exists(ctx context.Context, specialistID, productID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&specialistProductModel{}).
		Where("specialist_id = ? AND product_id = ?", specialistID, productID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
