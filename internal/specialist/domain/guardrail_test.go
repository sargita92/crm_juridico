package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGuardrail_WithValidData_ReturnsGuardrail(t *testing.T) {
	g, err := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "Nao falar sobre precos", "Nao posso informar precos")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", g.ID)
	assert.Equal(t, GuardrailTypeForbiddenTopics, g.Type)
	assert.Equal(t, "Nao falar sobre precos", g.Rule)
	assert.Equal(t, "Nao posso informar precos", g.Message)
	assert.True(t, g.Active)
	assert.False(t, g.CreatedAt.IsZero())
}

func TestNewGuardrail_EmptyRule_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "", "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleRequired)
}

func TestNewGuardrail_RuleTooLong_ReturnsError(t *testing.T) {
	longRule := strings.Repeat("a", MaxGuardrailRuleLength+1)
	_, err := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, longRule, "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleTooLong)
}

func TestNewGuardrail_MessageTooLong_ReturnsError(t *testing.T) {
	longMsg := strings.Repeat("a", MaxGuardrailMessageLength+1)
	_, err := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", longMsg)
	assert.ErrorIs(t, err, ErrGuardrailMessageTooLong)
}

func TestNewGuardrail_InvalidType_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "nome-teste", "invalid_type", "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailTypeInvalid)
}

func TestNewGuardrail_AllValidTypes(t *testing.T) {
	for _, gType := range []GuardrailType{
		GuardrailTypeForbiddenTopics,
		GuardrailTypeScopeLimit,
		GuardrailTypeResponseTone,
		GuardrailTypeSecurityLGPD,
		GuardrailTypeHumanEscalation,
		GuardrailTypeOutputValidation,
	} {
		t.Run(string(gType), func(t *testing.T) {
			g, err := NewGuardrail("uuid-1", "nome-teste", gType, "regra", "msg")
			require.NoError(t, err)
			assert.Equal(t, gType, g.Type)
		})
	}
}

// Regressão: Name é obrigatório e faz parte da validação no construtor +
// no Update — motivação é auditoria (busca fácil por nome).
func TestNewGuardrail_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "", GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailNameRequired)
}

func TestNewGuardrail_NameTooLong_ReturnsError(t *testing.T) {
	longName := string(make([]byte, MaxGuardrailNameLength+1))
	_, err := NewGuardrail("uuid-1", longName, GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailNameTooLong)
}

func TestGuardrail_Update_EmptyName_ReturnsError(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", "msg")
	err := g.Update("", GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailNameRequired)
	assert.Equal(t, "nome-teste", g.Name, "estado não deve mudar em erro")
}

func TestNewGuardrail_EmptyMessage_ReturnsGuardrail(t *testing.T) {
	g, err := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, err)
	assert.Empty(t, g.Message)
}

func TestGuardrail_Update_Success(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra antiga", "msg antiga")

	err := g.Update("nome-teste", GuardrailTypeScopeLimit, "regra nova", "msg nova")

	require.NoError(t, err)
	assert.Equal(t, GuardrailTypeScopeLimit, g.Type)
	assert.Equal(t, "regra nova", g.Rule)
	assert.Equal(t, "msg nova", g.Message)
}

func TestGuardrail_Update_EmptyRule_ReturnsError(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", "msg")
	err := g.Update("nome-teste", GuardrailTypeForbiddenTopics, "", "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleRequired)
	assert.Equal(t, "regra", g.Rule)
}

func TestGuardrail_Update_InvalidType_ReturnsError(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", "msg")
	err := g.Update("nome-teste", "invalid", "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailTypeInvalid)
}

func TestGuardrail_Toggle(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "nome-teste", GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.True(t, g.Active)

	g.Toggle()
	assert.False(t, g.Active)

	g.Toggle()
	assert.True(t, g.Active)
}
