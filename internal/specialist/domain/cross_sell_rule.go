package domain

import (
	"regexp"
	"strings"
	"time"
)

type CrossSellTriggerType string

const (
	CrossSellTriggerKeyword    CrossSellTriggerType = "keyword"
	CrossSellTriggerStepAnswer CrossSellTriggerType = "step_answer"
)

type KeywordTrigger struct {
	Termos []string `json:"termos"`
}

type StepAnswerTrigger struct {
	StepID string `json:"step_id"`
	Regex  string `json:"regex"`
}

type CrossSellRule struct {
	ID              string
	SpecialistID    string
	Ordem           int
	TriggerType     CrossSellTriggerType
	TriggerConfig   any
	TargetProductID string
	Ativo           bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func NewCrossSellRule(
	id, specialistID string,
	ordem int,
	triggerType CrossSellTriggerType,
	triggerConfig any,
	targetProductID string,
) (*CrossSellRule, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	if targetProductID == "" {
		return nil, ErrTargetProductRequired
	}
	if err := validateTriggerConfig(triggerType, triggerConfig); err != nil {
		return nil, err
	}
	now := time.Now()
	return &CrossSellRule{
		ID:              id,
		SpecialistID:    specialistID,
		Ordem:           ordem,
		TriggerType:     triggerType,
		TriggerConfig:   triggerConfig,
		TargetProductID: targetProductID,
		Ativo:           true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// ValidateTriggerConfig is the exported form of validateTriggerConfig,
// used by application use cases to validate before persisting updates.
func ValidateTriggerConfig(t CrossSellTriggerType, cfg any) error {
	return validateTriggerConfig(t, cfg)
}

func validateTriggerConfig(t CrossSellTriggerType, cfg any) error {
	switch t {
	case CrossSellTriggerKeyword:
		kw, ok := cfg.(KeywordTrigger)
		if !ok || len(kw.Termos) == 0 {
			return ErrKeywordTriggerEmpty
		}
		for _, term := range kw.Termos {
			if strings.TrimSpace(term) == "" {
				return ErrKeywordTriggerEmpty
			}
		}
	case CrossSellTriggerStepAnswer:
		sa, ok := cfg.(StepAnswerTrigger)
		if !ok || sa.StepID == "" {
			return ErrStepAnswerTriggerInvalid
		}
		if _, err := regexp.Compile(sa.Regex); err != nil {
			return ErrInvalidRegex
		}
	default:
		return ErrUnsupportedTrigger
	}
	return nil
}
