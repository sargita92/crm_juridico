package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type GormPhoneNumberRepository struct {
	db *gorm.DB
}

func NewGormPhoneNumberRepository(db *gorm.DB) *GormPhoneNumberRepository {
	return &GormPhoneNumberRepository{db: db}
}

func (r *GormPhoneNumberRepository) Create(ctx context.Context, pn *domain.ProductPhoneNumber) error {
	model := phoneNumberToModel(pn)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormPhoneNumberRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&phoneNumberModel{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *GormPhoneNumberRepository) FindByPhoneNumber(ctx context.Context, phoneNumber string) (*domain.ProductPhoneNumber, error) {
	var model phoneNumberModel
	if err := r.db.WithContext(ctx).Where("phone_number = ?", phoneNumber).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return phoneNumberToDomain(&model), nil
}

func (r *GormPhoneNumberRepository) FindByProductID(ctx context.Context, productID string) ([]domain.ProductPhoneNumber, error) {
	var models []phoneNumberModel
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]domain.ProductPhoneNumber, len(models))
	for i := range models {
		result[i] = *phoneNumberToDomain(&models[i])
	}
	return result, nil
}
