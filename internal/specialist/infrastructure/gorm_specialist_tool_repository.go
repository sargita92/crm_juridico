package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type specialistToolModel struct {
	ID           string    `gorm:"column:id;primaryKey;type:char(36)"`
	SpecialistID string    `gorm:"column:specialist_id;type:char(36)"`
	ToolName     string    `gorm:"column:tool_name"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistToolModel) TableName() string { return "specialist_tools" }

type GormSpecialistToolRepository struct {
	db *gorm.DB
}

func NewGormSpecialistToolRepository(db *gorm.DB) *GormSpecialistToolRepository {
	return &GormSpecialistToolRepository{db: db}
}

func (r *GormSpecialistToolRepository) Associate(ctx context.Context, specialistID, toolName string) error {
	model := specialistToolModel{
		ID:           uuid.New().String(),
		SpecialistID: specialistID,
		ToolName:     toolName,
		CreatedAt:    time.Now(),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.ErrToolAlreadyAssociated
		}
		return err
	}
	return nil
}

func (r *GormSpecialistToolRepository) Dissociate(ctx context.Context, specialistID, toolName string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND tool_name = ?", specialistID, toolName).
		Delete(&specialistToolModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrToolNotAssociated
	}
	return nil
}

func (r *GormSpecialistToolRepository) FindToolNamesBySpecialistID(ctx context.Context, specialistID string) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).
		Model(&specialistToolModel{}).
		Where("specialist_id = ?", specialistID).
		Pluck("tool_name", &names).Error
	return names, err
}

func (r *GormSpecialistToolRepository) FindAll(ctx context.Context, specialistID string) ([]domain.SpecialistTool, error) {
	var models []specialistToolModel
	err := r.db.WithContext(ctx).
		Where("specialist_id = ?", specialistID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.SpecialistTool, len(models))
	for i, m := range models {
		result[i] = domain.SpecialistTool{
			ID:           m.ID,
			SpecialistID: m.SpecialistID,
			ToolName:     m.ToolName,
			CreatedAt:    m.CreatedAt,
		}
	}
	return result, nil
}
