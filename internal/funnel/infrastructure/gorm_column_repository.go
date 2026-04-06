package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GormColumnRepository struct {
	db *gorm.DB
}

func NewGormColumnRepository(db *gorm.DB) *GormColumnRepository {
	return &GormColumnRepository{db: db}
}

func (r *GormColumnRepository) Create(ctx context.Context, col *domain.Column) error {
	model := columnToModel(col)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormColumnRepository) FindByID(ctx context.Context, id string) (*domain.Column, error) {
	var model columnModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrColumnNotFound
		}
		return nil, err
	}
	return columnToDomain(&model), nil
}

func (r *GormColumnRepository) Update(ctx context.Context, col *domain.Column) error {
	model := columnToModel(col)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrColumnNotFound
	}
	return nil
}

func (r *GormColumnRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&columnModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrColumnNotFound
	}
	return nil
}

func (r *GormColumnRepository) FindByFunnelID(ctx context.Context, funnelID string) ([]domain.Column, error) {
	var models []columnModel
	if err := r.db.WithContext(ctx).
		Where("funnel_id = ?", funnelID).
		Order("order_index ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	cols := make([]domain.Column, len(models))
	for i := range models {
		cols[i] = *columnToDomain(&models[i])
	}
	return cols, nil
}

func (r *GormColumnRepository) FindEntryByFunnelID(ctx context.Context, funnelID string) (*domain.Column, error) {
	var model columnModel
	if err := r.db.WithContext(ctx).
		Where("funnel_id = ? AND type = ?", funnelID, string(domain.ColumnTypeEntry)).
		Order("order_index ASC").
		Limit(1).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrColumnNotFound
		}
		return nil, err
	}
	return columnToDomain(&model), nil
}

func (r *GormColumnRepository) CountByFunnelID(ctx context.Context, funnelID string) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&columnModel{}).
		Where("funnel_id = ?", funnelID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *GormColumnRepository) GetMaxOrderIndex(ctx context.Context, funnelID string) (int, error) {
	var maxOrder *int
	if err := r.db.WithContext(ctx).
		Model(&columnModel{}).
		Where("funnel_id = ?", funnelID).
		Select("MAX(order_index)").
		Scan(&maxOrder).Error; err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return -1, nil
	}
	return *maxOrder, nil
}

func (r *GormColumnRepository) SwapOrder(ctx context.Context, col1ID string, order1 int, col2ID string, order2 int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&columnModel{}).
			Where("id = ?", col1ID).
			Update("order_index", order1).Error; err != nil {
			return err
		}
		if err := tx.Model(&columnModel{}).
			Where("id = ?", col2ID).
			Update("order_index", order2).Error; err != nil {
			return err
		}
		return nil
	})
}
