package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type stepModel struct {
	ID             string `gorm:"primaryKey;column:id;type:char(36)"`
	SpecialistID   string `gorm:"column:specialist_id;type:char(36);not null"`
	OrderIndex     int    `gorm:"column:order_index;not null"`
	Text           string `gorm:"column:text;type:text;not null"`
	DataType       string `gorm:"column:data_type;not null;default:free_text"`
	Required       bool   `gorm:"column:required;not null;default:true"`
	Score          int    `gorm:"column:score;not null;default:0"`
	TargetColumnID string `gorm:"column:target_column_id;type:char(36)"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (stepModel) TableName() string { return "steps" }

func stepToModel(s *domain.Step) *stepModel {
	return &stepModel{
		ID: s.ID, SpecialistID: s.SpecialistID, OrderIndex: s.OrderIndex,
		Text: s.Text, DataType: string(s.DataType), Required: s.Required, Score: s.Score,
		TargetColumnID: s.TargetColumnID,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}

func stepToDomain(m *stepModel) *domain.Step {
	return &domain.Step{
		ID: m.ID, SpecialistID: m.SpecialistID, OrderIndex: m.OrderIndex,
		Text: m.Text, DataType: domain.StepDataType(m.DataType), Required: m.Required, Score: m.Score,
		TargetColumnID: m.TargetColumnID,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type GormStepRepository struct {
	db *gorm.DB
}

func NewGormStepRepository(db *gorm.DB) *GormStepRepository {
	return &GormStepRepository{db: db}
}

func (r *GormStepRepository) Create(ctx context.Context, step *domain.Step) error {
	return r.db.WithContext(ctx).Create(stepToModel(step)).Error
}

func (r *GormStepRepository) FindByID(ctx context.Context, id string) (*domain.Step, error) {
	var model stepModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrStepNotFound
		}
		return nil, err
	}
	return stepToDomain(&model), nil
}

func (r *GormStepRepository) Update(ctx context.Context, step *domain.Step) error {
	result := r.db.WithContext(ctx).Save(stepToModel(step))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrStepNotFound
	}
	return nil
}

func (r *GormStepRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&stepModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrStepNotFound
	}
	return nil
}

func (r *GormStepRepository) FindBySpecialistID(ctx context.Context, specialistID string) ([]domain.Step, error) {
	var models []stepModel
	if err := r.db.WithContext(ctx).Where("specialist_id = ?", specialistID).Order("order_index ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	steps := make([]domain.Step, len(models))
	for i := range models {
		steps[i] = *stepToDomain(&models[i])
	}
	return steps, nil
}

func (r *GormStepRepository) GetMaxOrderIndex(ctx context.Context, specialistID string) (int, error) {
	var maxOrder *int
	err := r.db.WithContext(ctx).Model(&stepModel{}).Where("specialist_id = ?", specialistID).Select("MAX(order_index)").Scan(&maxOrder).Error
	if err != nil {
		return 0, err
	}
	if maxOrder == nil {
		return 0, nil
	}
	return *maxOrder, nil
}

func (r *GormStepRepository) ReorderAfterDelete(ctx context.Context, specialistID string, deletedOrder int) error {
	return r.db.WithContext(ctx).
		Model(&stepModel{}).
		Where("specialist_id = ? AND order_index > ?", specialistID, deletedOrder).
		UpdateColumn("order_index", gorm.Expr("order_index - 1")).Error
}

func (r *GormStepRepository) SwapOrder(ctx context.Context, step1ID string, order1 int, step2ID string, order2 int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&stepModel{}).Where("id = ?", step1ID).UpdateColumn("order_index", order2).Error; err != nil {
			return err
		}
		return tx.Model(&stepModel{}).Where("id = ?", step2ID).UpdateColumn("order_index", order1).Error
	})
}
