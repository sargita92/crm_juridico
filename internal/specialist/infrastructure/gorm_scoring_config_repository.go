package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type scoringConfigModel struct {
	ID           string `gorm:"primaryKey;column:id;type:char(36)"`
	SpecialistID string `gorm:"column:specialist_id;type:char(36);not null;uniqueIndex"`
	Threshold    int    `gorm:"column:threshold;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (scoringConfigModel) TableName() string { return "scoring_configs" }

func scoringToModel(sc *domain.ScoringConfig) *scoringConfigModel {
	return &scoringConfigModel{
		ID: sc.ID, SpecialistID: sc.SpecialistID, Threshold: sc.Threshold,
		CreatedAt: sc.CreatedAt, UpdatedAt: sc.UpdatedAt,
	}
}

func scoringToDomain(m *scoringConfigModel) *domain.ScoringConfig {
	return &domain.ScoringConfig{
		ID: m.ID, SpecialistID: m.SpecialistID, Threshold: m.Threshold,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type GormScoringConfigRepository struct {
	db *gorm.DB
}

func NewGormScoringConfigRepository(db *gorm.DB) *GormScoringConfigRepository {
	return &GormScoringConfigRepository{db: db}
}

func (r *GormScoringConfigRepository) CreateOrUpdate(ctx context.Context, config *domain.ScoringConfig) error {
	model := scoringToModel(config)
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *GormScoringConfigRepository) FindBySpecialistID(ctx context.Context, specialistID string) (*domain.ScoringConfig, error) {
	var model scoringConfigModel
	if err := r.db.WithContext(ctx).Where("specialist_id = ?", specialistID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrScoringConfigNotFound
		}
		return nil, err
	}
	return scoringToDomain(&model), nil
}
