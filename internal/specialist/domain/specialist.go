package domain

import (
	"errors"
	"time"
)

type SpecialistStatus string

const (
	SpecialistStatusActive   SpecialistStatus = "active"
	SpecialistStatusInactive SpecialistStatus = "inactive"
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
)

type Specialist struct {
	ID          string
	Name        string
	Description string
	Prompt      string
	Status      SpecialistStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
		ID:          id,
		Name:        name,
		Description: description,
		Prompt:      prompt,
		Status:      SpecialistStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
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
