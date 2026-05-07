package application

import (
	"testing"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
)

func makeGuardrail(gType specDomain.GuardrailType, rule, message string, active bool) specDomain.Guardrail {
	return specDomain.Guardrail{
		ID:      "g1",
		Type:    gType,
		Rule:    rule,
		Message: message,
		Active:  active,
	}
}

func TestGuardrailChecker_NoViolation(t *testing.T) {
	checker := NewGuardrailChecker()
	guardrails := []specDomain.Guardrail{
		makeGuardrail(specDomain.GuardrailTypeForbiddenTopics, "politica, religiao", "Nao posso falar sobre isso.", true),
	}

	violated, fallback := checker.Check("Olha, o seguro cobre varios riscos.", guardrails)
	assert.False(t, violated)
	assert.Equal(t, "", fallback)
}

func TestGuardrailChecker_ViolationFound(t *testing.T) {
	checker := NewGuardrailChecker()
	guardrails := []specDomain.Guardrail{
		makeGuardrail(specDomain.GuardrailTypeForbiddenTopics, "politica, religiao", "Nao posso falar sobre isso.", true),
	}

	violated, fallback := checker.Check("Vamos falar sobre politica hoje.", guardrails)
	assert.True(t, violated)
	assert.Equal(t, "Nao posso falar sobre isso.", fallback)
}

func TestGuardrailChecker_InactiveGuardrailIgnored(t *testing.T) {
	checker := NewGuardrailChecker()
	guardrails := []specDomain.Guardrail{
		makeGuardrail(specDomain.GuardrailTypeForbiddenTopics, "politica", "Nao posso falar sobre isso.", false),
	}

	violated, fallback := checker.Check("Vamos falar sobre politica.", guardrails)
	assert.False(t, violated)
	assert.Equal(t, "", fallback)
}

func TestGuardrailChecker_ScopeLimitTypeIgnored(t *testing.T) {
	checker := NewGuardrailChecker()
	guardrails := []specDomain.Guardrail{
		makeGuardrail(specDomain.GuardrailTypeScopeLimit, "esports", "Fora do escopo.", true),
	}

	violated, fallback := checker.Check("Falando sobre esports aqui.", guardrails)
	assert.False(t, violated)
	assert.Equal(t, "", fallback)
}

func TestGuardrailChecker_CaseInsensitive(t *testing.T) {
	checker := NewGuardrailChecker()
	guardrails := []specDomain.Guardrail{
		makeGuardrail(specDomain.GuardrailTypeForbiddenTopics, "POLITICA", "Nao posso falar sobre isso.", true),
	}

	violated, _ := checker.Check("Vamos falar sobre Politica.", guardrails)
	assert.True(t, violated)
}
