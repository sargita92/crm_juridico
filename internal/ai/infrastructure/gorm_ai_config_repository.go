package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type aiConfigModel struct {
	ID              string  `gorm:"primaryKey;column:id;type:char(36)"`
	SpecialistID    string  `gorm:"column:specialist_id;type:char(36)"`
	Provider        string  `gorm:"column:provider;type:varchar(50);not null"`
	Model           string  `gorm:"column:model;type:varchar(100);not null"`
	Temperature     float64 `gorm:"column:temperature;type:decimal(3,2);not null"`
	MaxTokens       int     `gorm:"column:max_tokens;not null"`
	DebounceSeconds int     `gorm:"column:debounce_seconds;not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (aiConfigModel) TableName() string { return "ai_configs" }

func aiConfigToModel(c *domain.AIConfig) *aiConfigModel {
	return &aiConfigModel{
		ID:              c.ID,
		SpecialistID:    c.SpecialistID,
		Provider:        c.Provider,
		Model:           c.Model,
		Temperature:     c.Temperature,
		MaxTokens:       c.MaxTokens,
		DebounceSeconds: c.DebounceSeconds,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func aiConfigToDomain(m *aiConfigModel) *domain.AIConfig {
	return &domain.AIConfig{
		ID:              m.ID,
		SpecialistID:    m.SpecialistID,
		Provider:        m.Provider,
		Model:           m.Model,
		Temperature:     m.Temperature,
		MaxTokens:       m.MaxTokens,
		DebounceSeconds: m.DebounceSeconds,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// GormAIConfigRepository implements domain.AIConfigRepository using GORM.
type GormAIConfigRepository struct {
	db *gorm.DB
}

// NewGormAIConfigRepository creates a new GormAIConfigRepository.
func NewGormAIConfigRepository(db *gorm.DB) *GormAIConfigRepository {
	return &GormAIConfigRepository{db: db}
}

// CreateOrUpdate inserts or updates an AIConfig record.
func (r *GormAIConfigRepository) CreateOrUpdate(ctx context.Context, config *domain.AIConfig) error {
	model := aiConfigToModel(config)
	return r.db.WithContext(ctx).Save(model).Error
}

// FindBySpecialistID retrieves an AIConfig for the given specialist.
func (r *GormAIConfigRepository) FindBySpecialistID(ctx context.Context, specialistID string) (*domain.AIConfig, error) {
	var model aiConfigModel
	if err := r.db.WithContext(ctx).Where("specialist_id = ?", specialistID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAIConfigNotFound
		}
		return nil, err
	}
	return aiConfigToDomain(&model), nil
}

// FindGlobal retrieves the global AIConfig (no specialist association).
func (r *GormAIConfigRepository) FindGlobal(ctx context.Context) (*domain.AIConfig, error) {
	var model aiConfigModel
	if err := r.db.WithContext(ctx).Where("specialist_id = '' OR specialist_id IS NULL").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAIConfigNotFound
		}
		return nil, err
	}
	return aiConfigToDomain(&model), nil
}
