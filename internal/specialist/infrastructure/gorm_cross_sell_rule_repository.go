package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type crossSellRuleModel struct {
	ID              string `gorm:"primaryKey;column:id;type:varchar(36)"`
	SpecialistID    string `gorm:"column:specialist_id;type:varchar(36);index"`
	Ordem           int    `gorm:"column:ordem"`
	TriggerType     string `gorm:"column:trigger_type;type:varchar(20)"`
	TriggerConfig   string `gorm:"column:trigger_config;type:json"`
	TargetProductID string `gorm:"column:target_product_id;type:varchar(36);index"`
	Ativo           bool   `gorm:"column:ativo"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (crossSellRuleModel) TableName() string { return "cross_sell_rules" }

func crossSellRuleToModel(r *domain.CrossSellRule) (*crossSellRuleModel, error) {
	raw, err := json.Marshal(r.TriggerConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal trigger_config: %w", err)
	}
	return &crossSellRuleModel{
		ID:              r.ID,
		SpecialistID:    r.SpecialistID,
		Ordem:           r.Ordem,
		TriggerType:     string(r.TriggerType),
		TriggerConfig:   string(raw),
		TargetProductID: r.TargetProductID,
		Ativo:           r.Ativo,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}, nil
}

func crossSellRuleToDomain(m *crossSellRuleModel) (*domain.CrossSellRule, error) {
	triggerType := domain.CrossSellTriggerType(m.TriggerType)
	var triggerConfig any
	switch triggerType {
	case domain.CrossSellTriggerKeyword:
		var kw domain.KeywordTrigger
		if err := json.Unmarshal([]byte(m.TriggerConfig), &kw); err != nil {
			return nil, fmt.Errorf("unmarshal keyword trigger: %w", err)
		}
		triggerConfig = kw
	case domain.CrossSellTriggerStepAnswer:
		var sa domain.StepAnswerTrigger
		if err := json.Unmarshal([]byte(m.TriggerConfig), &sa); err != nil {
			return nil, fmt.Errorf("unmarshal step_answer trigger: %w", err)
		}
		triggerConfig = sa
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedTrigger, m.TriggerType)
	}
	return &domain.CrossSellRule{
		ID:              m.ID,
		SpecialistID:    m.SpecialistID,
		Ordem:           m.Ordem,
		TriggerType:     triggerType,
		TriggerConfig:   triggerConfig,
		TargetProductID: m.TargetProductID,
		Ativo:           m.Ativo,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

type GormCrossSellRuleRepository struct {
	db *gorm.DB
}

func NewGormCrossSellRuleRepository(db *gorm.DB) *GormCrossSellRuleRepository {
	return &GormCrossSellRuleRepository{db: db}
}

func (r *GormCrossSellRuleRepository) Save(ctx context.Context, rule *domain.CrossSellRule) error {
	model, err := crossSellRuleToModel(rule)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *GormCrossSellRuleRepository) FindByID(ctx context.Context, id string) (*domain.CrossSellRule, error) {
	var model crossSellRuleModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrCrossSellRuleNotFound
		}
		return nil, err
	}
	return crossSellRuleToDomain(&model)
}

func (r *GormCrossSellRuleRepository) ListBySpecialistID(ctx context.Context, specialistID string) ([]*domain.CrossSellRule, error) {
	var models []crossSellRuleModel
	if err := r.db.WithContext(ctx).
		Where("specialist_id = ?", specialistID).
		Order("ordem ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return modelsToRules(models)
}

func (r *GormCrossSellRuleRepository) ListActiveBySpecialistOrdered(ctx context.Context, specialistID string) ([]*domain.CrossSellRule, error) {
	var models []crossSellRuleModel
	if err := r.db.WithContext(ctx).
		Where("specialist_id = ? AND ativo = ?", specialistID, true).
		Order("ordem ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return modelsToRules(models)
}

func (r *GormCrossSellRuleRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&crossSellRuleModel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrCrossSellRuleNotFound
	}
	return nil
}

func modelsToRules(models []crossSellRuleModel) ([]*domain.CrossSellRule, error) {
	rules := make([]*domain.CrossSellRule, 0, len(models))
	for i := range models {
		r, err := crossSellRuleToDomain(&models[i])
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}
