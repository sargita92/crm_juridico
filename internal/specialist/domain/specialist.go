package domain

import (
	"errors"
	"strings"
	"time"
)

type SpecialistStatus string

const (
	SpecialistStatusActive   SpecialistStatus = "active"
	SpecialistStatusInactive SpecialistStatus = "inactive"
)

type CrossSellMode string

const (
	CrossSellModeAnnounce CrossSellMode = "announce"
	CrossSellModeSilent   CrossSellMode = "silent"
	CrossSellModeConfirm  CrossSellMode = "confirm"
)

const MaxPromptLength = 10000
const MaxDescriptionLength = 500

var (
	ErrSpecialistNotFound        = errors.New("specialist not found")
	ErrSpecialistNameRequired    = errors.New("specialist name is required")
	ErrSpecialistPromptRequired  = errors.New("specialist prompt is required")
	ErrPromptTooLong             = errors.New("prompt exceeds maximum length")
	ErrDescriptionTooLong        = errors.New("description exceeds maximum length")
	ErrSpecialistAlreadyInactive = errors.New("specialist is already inactive")
	ErrSpecialistAlreadyActive   = errors.New("specialist is already active")
	ErrSpecialistInactive        = errors.New("specialist is inactive")
	ErrTenantAlreadyAssociated   = errors.New("tenant is already associated")
	ErrTenantNotAssociated       = errors.New("tenant is not associated")

	// Guardrail errors
	ErrSpecialistIDRequired    = errors.New("specialist ID is required")
	ErrGuardrailNotFound       = errors.New("guardrail not found")
	ErrGuardrailRuleRequired   = errors.New("guardrail rule is required")
	ErrGuardrailRuleTooLong    = errors.New("guardrail rule exceeds maximum length")
	ErrGuardrailMessageTooLong = errors.New("guardrail message exceeds maximum length")
	ErrGuardrailTypeInvalid    = errors.New("guardrail type is invalid")

	// Step errors
	ErrStepNotFound        = errors.New("step not found")
	ErrStepTextRequired    = errors.New("step text is required")
	ErrStepTextTooLong     = errors.New("step text exceeds maximum length")
	ErrStepDataTypeInvalid = errors.New("step data type is invalid")
	ErrStepScoreNegative   = errors.New("step score cannot be negative")

	// Scoring errors
	ErrScoringConfigNotFound        = errors.New("scoring config not found")
	ErrScoringThresholdInvalid      = errors.New("scoring threshold must be greater than zero")
	ErrScoringThresholdExceedsTotal = errors.New("scoring threshold exceeds total possible score")
	ErrHumanoMinNegative            = errors.New("scoring: threshold humano min cannot be negative")
	ErrHumanoMinAboveAprovado       = errors.New("scoring: threshold humano min cannot exceed threshold aprovado")

	// Cross-sell errors
	ErrCrossSellTemplateRequired = errors.New("specialist: cross-sell announce mode requires a template")

	// CrossSellRule errors
	ErrCrossSellRuleNotFound             = errors.New("cross-sell rule not found")
	ErrCrossSellRuleNotOwnedBySpecialist = errors.New("cross-sell rule does not belong to the given specialist")
	ErrTargetProductRequired             = errors.New("cross-sell rule: target product ID is required")
	ErrKeywordTriggerEmpty               = errors.New("cross-sell rule: keyword trigger must have at least one non-blank term")
	ErrStepAnswerTriggerInvalid          = errors.New("cross-sell rule: step_answer trigger requires a non-empty step ID")
	ErrInvalidRegex                      = errors.New("cross-sell rule: trigger regex is invalid")
	ErrUnsupportedTrigger                = errors.New("cross-sell rule: unsupported trigger type")
)

type Specialist struct {
	ID                            string
	Name                          string
	Description                   string
	Prompt                        string
	Status                        SpecialistStatus
	CreatedAt                     time.Time
	UpdatedAt                     time.Time
	CrossSellEnabled              bool
	CrossSellMode                 CrossSellMode
	CrossSellAnnouncementTemplate string
	AllowAICrossSellSuggestion    bool
}

func NewSpecialist(id, name, description, prompt string) (*Specialist, error) {
	if name == "" {
		return nil, ErrSpecialistNameRequired
	}
	if prompt == "" {
		return nil, ErrSpecialistPromptRequired
	}
	if len(prompt) > MaxPromptLength {
		return nil, ErrPromptTooLong
	}
	if len(description) > MaxDescriptionLength {
		return nil, ErrDescriptionTooLong
	}

	now := time.Now()
	return &Specialist{
		ID:               id,
		Name:             name,
		Description:      description,
		Prompt:           prompt,
		Status:           SpecialistStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		CrossSellMode:    CrossSellModeAnnounce,
		CrossSellEnabled: false,
	}, nil
}

func (s *Specialist) IsActive() bool {
	return s.Status == SpecialistStatusActive
}

func (s *Specialist) Update(name, description, prompt string) error {
	if name == "" {
		return ErrSpecialistNameRequired
	}
	if prompt == "" {
		return ErrSpecialistPromptRequired
	}
	if len(prompt) > MaxPromptLength {
		return ErrPromptTooLong
	}
	if len(description) > MaxDescriptionLength {
		return ErrDescriptionTooLong
	}
	s.Name = name
	s.Description = description
	s.Prompt = prompt
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Specialist) UpdatePrompt(prompt string) error {
	if prompt == "" {
		return ErrSpecialistPromptRequired
	}
	if len(prompt) > MaxPromptLength {
		return ErrPromptTooLong
	}
	s.Prompt = prompt
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Specialist) Deactivate() error {
	if s.Status == SpecialistStatusInactive {
		return ErrSpecialistAlreadyInactive
	}
	s.Status = SpecialistStatusInactive
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Specialist) Activate() error {
	if s.Status == SpecialistStatusActive {
		return ErrSpecialistAlreadyActive
	}
	s.Status = SpecialistStatusActive
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Specialist) EnableCrossSell(mode CrossSellMode, template string) error {
	if mode == CrossSellModeAnnounce && strings.TrimSpace(template) == "" {
		return ErrCrossSellTemplateRequired
	}
	s.CrossSellEnabled = true
	s.CrossSellMode = mode
	s.CrossSellAnnouncementTemplate = template
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Specialist) DisableCrossSell() {
	s.CrossSellEnabled = false
	s.UpdatedAt = time.Now()
}
