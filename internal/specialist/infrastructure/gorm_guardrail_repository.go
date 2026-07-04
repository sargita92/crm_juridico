package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type guardrailModel struct {
	ID           string `gorm:"primaryKey;column:id;type:char(36)"`
	SpecialistID string `gorm:"column:specialist_id;type:char(36);not null"`
	Name         string `gorm:"column:name;type:varchar(120);not null"`
	Type         string `gorm:"column:type;not null"`
	Rule         string `gorm:"column:rule;type:text;not null"`
	Message      string `gorm:"column:message;type:text"`
	Active       bool   `gorm:"column:active;not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (guardrailModel) TableName() string { return "guardrails" }

func guardrailToModel(g *domain.Guardrail) *guardrailModel {
	return &guardrailModel{
		ID: g.ID, SpecialistID: g.SpecialistID, Name: g.Name, Type: string(g.Type),
		Rule: g.Rule, Message: g.Message, Active: g.Active,
		CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}
}

func guardrailToDomain(m *guardrailModel) *domain.Guardrail {
	return &domain.Guardrail{
		ID: m.ID, SpecialistID: m.SpecialistID, Name: m.Name, Type: domain.GuardrailType(m.Type),
		Rule: m.Rule, Message: m.Message, Active: m.Active,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

type GormGuardrailRepository struct {
	db *gorm.DB
}

func NewGormGuardrailRepository(db *gorm.DB) *GormGuardrailRepository {
	return &GormGuardrailRepository{db: db}
}

func (r *GormGuardrailRepository) Create(ctx context.Context, guardrail *domain.Guardrail) error {
	return r.db.WithContext(ctx).Create(guardrailToModel(guardrail)).Error
}

func (r *GormGuardrailRepository) FindByID(ctx context.Context, id string) (*domain.Guardrail, error) {
	var model guardrailModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrGuardrailNotFound
		}
		return nil, err
	}
	return guardrailToDomain(&model), nil
}

func (r *GormGuardrailRepository) Update(ctx context.Context, guardrail *domain.Guardrail) error {
	result := r.db.WithContext(ctx).Save(guardrailToModel(guardrail))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGuardrailNotFound
	}
	return nil
}

func (r *GormGuardrailRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&guardrailModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrGuardrailNotFound
	}
	return nil
}

func (r *GormGuardrailRepository) FindBySpecialistID(ctx context.Context, specialistID string) ([]domain.Guardrail, error) {
	var models []guardrailModel
	if err := r.db.WithContext(ctx).Where("specialist_id = ?", specialistID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	guardrails := make([]domain.Guardrail, len(models))
	for i := range models {
		guardrails[i] = *guardrailToDomain(&models[i])
	}
	return guardrails, nil
}
