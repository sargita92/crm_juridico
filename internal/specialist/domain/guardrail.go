package domain

import "time"

type GuardrailType string

const (
	GuardrailTypeForbiddenTopics  GuardrailType = "forbidden_topics"
	GuardrailTypeScopeLimit       GuardrailType = "scope_limit"
	GuardrailTypeResponseTone     GuardrailType = "response_tone"
	GuardrailTypeSecurityLGPD     GuardrailType = "security_lgpd"
	GuardrailTypeHumanEscalation  GuardrailType = "human_escalation"
	GuardrailTypeOutputValidation GuardrailType = "output_validation"
)

const (
	MaxGuardrailNameLength    = 120
	MaxGuardrailRuleLength    = 2000
	MaxGuardrailMessageLength = 1000
)

type Guardrail struct {
	ID           string
	SpecialistID string
	Name         string
	Type         GuardrailType
	Rule         string
	Message      string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewGuardrail(id, specialistID, name string, gType GuardrailType, rule, message string) (*Guardrail, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	if name == "" {
		return nil, ErrGuardrailNameRequired
	}
	if len(name) > MaxGuardrailNameLength {
		return nil, ErrGuardrailNameTooLong
	}
	if rule == "" {
		return nil, ErrGuardrailRuleRequired
	}
	if len(rule) > MaxGuardrailRuleLength {
		return nil, ErrGuardrailRuleTooLong
	}
	if len(message) > MaxGuardrailMessageLength {
		return nil, ErrGuardrailMessageTooLong
	}
	if !isValidGuardrailType(gType) {
		return nil, ErrGuardrailTypeInvalid
	}

	now := time.Now()
	return &Guardrail{
		ID: id, SpecialistID: specialistID, Name: name, Type: gType,
		Rule: rule, Message: message, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (g *Guardrail) Update(name string, gType GuardrailType, rule, message string) error {
	if name == "" {
		return ErrGuardrailNameRequired
	}
	if len(name) > MaxGuardrailNameLength {
		return ErrGuardrailNameTooLong
	}
	if rule == "" {
		return ErrGuardrailRuleRequired
	}
	if len(rule) > MaxGuardrailRuleLength {
		return ErrGuardrailRuleTooLong
	}
	if len(message) > MaxGuardrailMessageLength {
		return ErrGuardrailMessageTooLong
	}
	if !isValidGuardrailType(gType) {
		return ErrGuardrailTypeInvalid
	}
	g.Name = name
	g.Type = gType
	g.Rule = rule
	g.Message = message
	g.UpdatedAt = time.Now()
	return nil
}

func (g *Guardrail) Toggle() {
	g.Active = !g.Active
	g.UpdatedAt = time.Now()
}

func isValidGuardrailType(t GuardrailType) bool {
	switch t {
	case GuardrailTypeForbiddenTopics,
		GuardrailTypeScopeLimit,
		GuardrailTypeResponseTone,
		GuardrailTypeSecurityLGPD,
		GuardrailTypeHumanEscalation,
		GuardrailTypeOutputValidation:
		return true
	}
	return false
}
