package domain

import "time"

type GuardrailType string

const (
	GuardrailTypeForbiddenTopics GuardrailType = "forbidden_topics"
	GuardrailTypeScopeLimit      GuardrailType = "scope_limit"
	GuardrailTypeResponseTone    GuardrailType = "response_tone"
)

const MaxGuardrailRuleLength = 2000
const MaxGuardrailMessageLength = 1000

type Guardrail struct {
	ID           string
	SpecialistID string
	Type         GuardrailType
	Rule         string
	Message      string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewGuardrail(id, specialistID string, gType GuardrailType, rule, message string) (*Guardrail, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
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
		ID: id, SpecialistID: specialistID, Type: gType,
		Rule: rule, Message: message, Active: true,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (g *Guardrail) Update(gType GuardrailType, rule, message string) error {
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
	case GuardrailTypeForbiddenTopics, GuardrailTypeScopeLimit, GuardrailTypeResponseTone:
		return true
	}
	return false
}
