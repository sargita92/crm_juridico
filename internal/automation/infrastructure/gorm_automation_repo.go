package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// GormAutomationRepository is the GORM-backed implementation of domain.AutomationRepository.
type GormAutomationRepository struct {
	db *gorm.DB
}

func NewGormAutomationRepository(db *gorm.DB) *GormAutomationRepository {
	return &GormAutomationRepository{db: db}
}

func (r *GormAutomationRepository) Create(ctx context.Context, a *domain.Automation) error {
	return r.db.WithContext(ctx).Create(automationToModel(a)).Error
}

func (r *GormAutomationRepository) FindByID(ctx context.Context, id string) (*domain.Automation, error) {
	var m automationModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrAutomationNotFound
		}
		return nil, err
	}
	return automationToDomain(&m), nil
}

func (r *GormAutomationRepository) Update(ctx context.Context, a *domain.Automation) error {
	result := r.db.WithContext(ctx).Save(automationToModel(a))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAutomationNotFound
	}
	return nil
}

func (r *GormAutomationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&automationModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrAutomationNotFound
	}
	return nil
}

// FindByTenantAndColumn returns active automations for the given tenant and column,
// ordered by priority ascending.
func (r *GormAutomationRepository) FindByTenantAndColumn(ctx context.Context, tenantID, columnID string) ([]domain.Automation, error) {
	var models []automationModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND column_id = ? AND active = 1", tenantID, columnID).
		Order("priority ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return modelsToAutomations(models), nil
}

// FindByFunnelID returns all automations for the given tenant and funnel,
// ordered by priority ascending.
func (r *GormAutomationRepository) FindByFunnelID(ctx context.Context, tenantID, funnelID string) ([]domain.Automation, error) {
	var models []automationModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND funnel_id = ?", tenantID, funnelID).
		Order("priority ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return modelsToAutomations(models), nil
}

// FindActiveByType returns all active automations of the given type.
func (r *GormAutomationRepository) FindActiveByType(ctx context.Context, automationType domain.AutomationType) ([]domain.Automation, error) {
	var models []automationModel
	if err := r.db.WithContext(ctx).
		Where("type = ? AND active = 1", string(automationType)).
		Find(&models).Error; err != nil {
		return nil, err
	}
	return modelsToAutomations(models), nil
}

func modelsToAutomations(models []automationModel) []domain.Automation {
	result := make([]domain.Automation, len(models))
	for i := range models {
		result[i] = *automationToDomain(&models[i])
	}
	return result
}
