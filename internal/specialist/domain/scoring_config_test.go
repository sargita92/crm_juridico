package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScoringConfig_WithValidData_ReturnsConfig(t *testing.T) {
	sc, err := NewScoringConfig("uuid-1", "spec-1", 70)

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", sc.ID)
	assert.Equal(t, "spec-1", sc.SpecialistID)
	assert.Equal(t, 70, sc.Threshold)
	assert.False(t, sc.CreatedAt.IsZero())
}

func TestNewScoringConfig_EmptySpecialistID_ReturnsError(t *testing.T) {
	_, err := NewScoringConfig("uuid-1", "", 70)
	assert.ErrorIs(t, err, ErrSpecialistIDRequired)
}

func TestNewScoringConfig_ZeroThreshold_ReturnsError(t *testing.T) {
	_, err := NewScoringConfig("uuid-1", "spec-1", 0)
	assert.ErrorIs(t, err, ErrScoringThresholdInvalid)
}

func TestNewScoringConfig_NegativeThreshold_ReturnsError(t *testing.T) {
	_, err := NewScoringConfig("uuid-1", "spec-1", -10)
	assert.ErrorIs(t, err, ErrScoringThresholdInvalid)
}

func TestScoringConfig_UpdateThreshold_Success(t *testing.T) {
	sc, _ := NewScoringConfig("uuid-1", "spec-1", 70)

	err := sc.UpdateThreshold(50, 100)

	require.NoError(t, err)
	assert.Equal(t, 50, sc.Threshold)
}

func TestScoringConfig_UpdateThreshold_ExceedsTotal_ReturnsError(t *testing.T) {
	sc, _ := NewScoringConfig("uuid-1", "spec-1", 70)

	err := sc.UpdateThreshold(80, 60)

	assert.ErrorIs(t, err, ErrScoringThresholdExceedsTotal)
	assert.Equal(t, 70, sc.Threshold)
}

func TestScoringConfig_UpdateThreshold_Zero_ReturnsError(t *testing.T) {
	sc, _ := NewScoringConfig("uuid-1", "spec-1", 70)

	err := sc.UpdateThreshold(0, 100)

	assert.ErrorIs(t, err, ErrScoringThresholdInvalid)
}

func TestScoringConfig_HumanoMinDefaultsToZero(t *testing.T) {
	sc, err := NewScoringConfig("id", "spec-1", 80)
	require.NoError(t, err)
	assert.Equal(t, 0, sc.ThresholdHumanoMin)
}

func TestScoringConfig_UpdateThresholdHumanoMin_ValidRange(t *testing.T) {
	sc, _ := NewScoringConfig("id", "spec-1", 80)
	err := sc.UpdateThresholdHumanoMin(50, 100)
	require.NoError(t, err)
	assert.Equal(t, 50, sc.ThresholdHumanoMin)
}

func TestScoringConfig_UpdateThresholdHumanoMin_RejectsAboveAprovado(t *testing.T) {
	sc, _ := NewScoringConfig("id", "spec-1", 80)
	err := sc.UpdateThresholdHumanoMin(90, 100)
	require.ErrorIs(t, err, ErrHumanoMinAboveAprovado)
}

func TestScoringConfig_UpdateThresholdHumanoMin_RejectsNegative(t *testing.T) {
	sc, _ := NewScoringConfig("id", "spec-1", 80)
	err := sc.UpdateThresholdHumanoMin(-1, 100)
	require.ErrorIs(t, err, ErrHumanoMinNegative)
}
