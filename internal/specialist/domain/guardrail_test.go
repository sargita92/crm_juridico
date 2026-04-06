package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGuardrail_WithValidData_ReturnsGuardrail(t *testing.T) {
	g, err := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "Nao falar sobre precos", "Nao posso informar precos")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", g.ID)
	assert.Equal(t, "spec-1", g.SpecialistID)
	assert.Equal(t, GuardrailTypeForbiddenTopics, g.Type)
	assert.Equal(t, "Nao falar sobre precos", g.Rule)
	assert.Equal(t, "Nao posso informar precos", g.Message)
	assert.True(t, g.Active)
	assert.False(t, g.CreatedAt.IsZero())
}

func TestNewGuardrail_EmptySpecialistID_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "", GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.ErrorIs(t, err, ErrSpecialistIDRequired)
}

func TestNewGuardrail_EmptyRule_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "", "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleRequired)
}

func TestNewGuardrail_RuleTooLong_ReturnsError(t *testing.T) {
	longRule := strings.Repeat("a", MaxGuardrailRuleLength+1)
	_, err := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, longRule, "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleTooLong)
}

func TestNewGuardrail_MessageTooLong_ReturnsError(t *testing.T) {
	longMsg := strings.Repeat("a", MaxGuardrailMessageLength+1)
	_, err := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra", longMsg)
	assert.ErrorIs(t, err, ErrGuardrailMessageTooLong)
}

func TestNewGuardrail_InvalidType_ReturnsError(t *testing.T) {
	_, err := NewGuardrail("uuid-1", "spec-1", "invalid_type", "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailTypeInvalid)
}

func TestNewGuardrail_AllValidTypes(t *testing.T) {
	for _, gType := range []GuardrailType{GuardrailTypeForbiddenTopics, GuardrailTypeScopeLimit, GuardrailTypeResponseTone} {
		t.Run(string(gType), func(t *testing.T) {
			g, err := NewGuardrail("uuid-1", "spec-1", gType, "regra", "msg")
			require.NoError(t, err)
			assert.Equal(t, gType, g.Type)
		})
	}
}

func TestNewGuardrail_EmptyMessage_ReturnsGuardrail(t *testing.T) {
	g, err := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra", "")
	require.NoError(t, err)
	assert.Empty(t, g.Message)
}

func TestGuardrail_Update_Success(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra antiga", "msg antiga")

	err := g.Update(GuardrailTypeScopeLimit, "regra nova", "msg nova")

	require.NoError(t, err)
	assert.Equal(t, GuardrailTypeScopeLimit, g.Type)
	assert.Equal(t, "regra nova", g.Rule)
	assert.Equal(t, "msg nova", g.Message)
}

func TestGuardrail_Update_EmptyRule_ReturnsError(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra", "msg")
	err := g.Update(GuardrailTypeForbiddenTopics, "", "msg")
	assert.ErrorIs(t, err, ErrGuardrailRuleRequired)
	assert.Equal(t, "regra", g.Rule)
}

func TestGuardrail_Update_InvalidType_ReturnsError(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra", "msg")
	err := g.Update("invalid", "regra", "msg")
	assert.ErrorIs(t, err, ErrGuardrailTypeInvalid)
}

func TestGuardrail_Toggle(t *testing.T) {
	g, _ := NewGuardrail("uuid-1", "spec-1", GuardrailTypeForbiddenTopics, "regra", "msg")
	assert.True(t, g.Active)

	g.Toggle()
	assert.False(t, g.Active)

	g.Toggle()
	assert.True(t, g.Active)
}
