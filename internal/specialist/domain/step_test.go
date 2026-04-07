package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStep_WithValidData_ReturnsStep(t *testing.T) {
	s, err := NewStep("uuid-1", "spec-1", 1, "Qual seu nome?", StepDataTypeFreeText, true, 10, "")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", s.ID)
	assert.Equal(t, "spec-1", s.SpecialistID)
	assert.Equal(t, 1, s.OrderIndex)
	assert.Equal(t, "Qual seu nome?", s.Text)
	assert.Equal(t, StepDataTypeFreeText, s.DataType)
	assert.True(t, s.Required)
	assert.Equal(t, 10, s.Score)
	assert.False(t, s.CreatedAt.IsZero())
}

func TestNewStep_EmptySpecialistID_ReturnsError(t *testing.T) {
	_, err := NewStep("uuid-1", "", 1, "texto", StepDataTypeFreeText, true, 0, "")
	assert.ErrorIs(t, err, ErrSpecialistIDRequired)
}

func TestNewStep_EmptyText_ReturnsError(t *testing.T) {
	_, err := NewStep("uuid-1", "spec-1", 1, "", StepDataTypeFreeText, true, 0, "")
	assert.ErrorIs(t, err, ErrStepTextRequired)
}

func TestNewStep_TextTooLong_ReturnsError(t *testing.T) {
	longText := strings.Repeat("a", MaxStepTextLength+1)
	_, err := NewStep("uuid-1", "spec-1", 1, longText, StepDataTypeFreeText, true, 0, "")
	assert.ErrorIs(t, err, ErrStepTextTooLong)
}

func TestNewStep_InvalidDataType_ReturnsError(t *testing.T) {
	_, err := NewStep("uuid-1", "spec-1", 1, "texto", "invalid", true, 0, "")
	assert.ErrorIs(t, err, ErrStepDataTypeInvalid)
}

func TestNewStep_NegativeScore_ReturnsError(t *testing.T) {
	_, err := NewStep("uuid-1", "spec-1", 1, "texto", StepDataTypeFreeText, true, -1, "")
	assert.ErrorIs(t, err, ErrStepScoreNegative)
}

func TestNewStep_ZeroScore_ReturnsStep(t *testing.T) {
	s, err := NewStep("uuid-1", "spec-1", 1, "texto", StepDataTypeFreeText, true, 0, "")
	require.NoError(t, err)
	assert.Equal(t, 0, s.Score)
}

func TestNewStep_AllValidDataTypes(t *testing.T) {
	for _, dt := range []StepDataType{StepDataTypeFreeText, StepDataTypeNumber, StepDataTypeDate, StepDataTypeDocument, StepDataTypeSelection} {
		t.Run(string(dt), func(t *testing.T) {
			s, err := NewStep("uuid-1", "spec-1", 1, "texto", dt, true, 0, "")
			require.NoError(t, err)
			assert.Equal(t, dt, s.DataType)
		})
	}
}

func TestStep_Update_Success(t *testing.T) {
	s, _ := NewStep("uuid-1", "spec-1", 1, "antigo", StepDataTypeFreeText, true, 10, "")

	err := s.Update("novo", StepDataTypeNumber, false, 20, "")

	require.NoError(t, err)
	assert.Equal(t, "novo", s.Text)
	assert.Equal(t, StepDataTypeNumber, s.DataType)
	assert.False(t, s.Required)
	assert.Equal(t, 20, s.Score)
}

func TestStep_Update_EmptyText_ReturnsError(t *testing.T) {
	s, _ := NewStep("uuid-1", "spec-1", 1, "texto", StepDataTypeFreeText, true, 0, "")
	err := s.Update("", StepDataTypeFreeText, true, 0, "")
	assert.ErrorIs(t, err, ErrStepTextRequired)
	assert.Equal(t, "texto", s.Text)
}

func TestStep_Update_NegativeScore_ReturnsError(t *testing.T) {
	s, _ := NewStep("uuid-1", "spec-1", 1, "texto", StepDataTypeFreeText, true, 10, "")
	err := s.Update("texto", StepDataTypeFreeText, true, -5, "")
	assert.ErrorIs(t, err, ErrStepScoreNegative)
	assert.Equal(t, 10, s.Score)
}
