package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConversationState_WithValidData_ReturnsState(t *testing.T) {
	state, err := NewConversationState("id-1", "conv-1", "spec-1")

	require.NoError(t, err)
	assert.Equal(t, "id-1", state.ID)
	assert.Equal(t, "conv-1", state.ConversationID)
	assert.Equal(t, "spec-1", state.SpecialistID)
	assert.Equal(t, 0, state.CurrentStepIndex)
	assert.NotNil(t, state.CollectedData)
	assert.Equal(t, 0, state.AccumulatedScore)
	assert.False(t, state.HandoffActive)
	assert.Nil(t, state.HandoffAt)
	assert.Nil(t, state.ResumedAt)
	assert.False(t, state.CreatedAt.IsZero())
	assert.False(t, state.UpdatedAt.IsZero())
}

func TestNewConversationState_EmptyConversationID_ReturnsError(t *testing.T) {
	_, err := NewConversationState("id-1", "", "spec-1")

	assert.ErrorIs(t, err, ErrConversationIDRequired)
}

func TestNewConversationState_EmptySpecialistID_ReturnsError(t *testing.T) {
	_, err := NewConversationState("id-1", "conv-1", "")

	assert.ErrorIs(t, err, ErrSpecialistIDRequired)
}

func TestConversationState_AdvanceStep_UpdatesState(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")

	state.AdvanceStep("name", "John", 10)

	assert.Equal(t, 1, state.CurrentStepIndex)
	assert.Equal(t, "John", state.CollectedData["name"])
	assert.Equal(t, 10, state.AccumulatedScore)
}

func TestConversationState_AdvanceStep_MultipleSteps_AccumulatesCorrectly(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")

	state.AdvanceStep("name", "John", 10)
	state.AdvanceStep("email", "john@example.com", 20)

	assert.Equal(t, 2, state.CurrentStepIndex)
	assert.Equal(t, "John", state.CollectedData["name"])
	assert.Equal(t, "john@example.com", state.CollectedData["email"])
	assert.Equal(t, 30, state.AccumulatedScore)
}

func TestConversationState_ActivateHandoff_SetsHandoffState(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")

	state.ActivateHandoff()

	assert.True(t, state.HandoffActive)
	require.NotNil(t, state.HandoffAt)
	assert.False(t, state.HandoffAt.IsZero())
}

func TestConversationState_DeactivateHandoff_SetsResumedState(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")
	state.ActivateHandoff()

	state.DeactivateHandoff()

	assert.False(t, state.HandoffActive)
	require.NotNil(t, state.ResumedAt)
	assert.False(t, state.ResumedAt.IsZero())
}

func TestConversationState_IsAfterResume_NoHandoff_ReturnsTrue(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")
	ts := time.Now().Add(-1 * time.Minute)

	result := state.IsAfterResume(ts)

	assert.True(t, result)
}

func TestConversationState_IsAfterResume_AfterResumedAt_ReturnsTrue(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")
	state.ActivateHandoff()
	state.DeactivateHandoff()
	tsAfter := time.Now().Add(1 * time.Minute)

	result := state.IsAfterResume(tsAfter)

	assert.True(t, result)
}

func TestConversationState_IsAfterResume_BeforeResumedAt_ReturnsFalse(t *testing.T) {
	state, _ := NewConversationState("id-1", "conv-1", "spec-1")
	state.ActivateHandoff()
	state.DeactivateHandoff()
	tsBefore := time.Now().Add(-1 * time.Minute)

	result := state.IsAfterResume(tsBefore)

	assert.False(t, result)
}

func TestConversationState_Reset(t *testing.T) {
	s, err := NewConversationState("id-1", "conv-1", "spec-1")
	require.NoError(t, err)

	s.AdvanceStep("nome", "Maria", 10)
	s.AdvanceStep("produto", "PROD_IDADE", 20)
	s.ActivateHandoff()

	before := s.UpdatedAt
	time.Sleep(time.Millisecond)
	s.Reset()

	assert.Equal(t, 0, s.CurrentStepIndex)
	assert.Equal(t, 0, s.AccumulatedScore)
	assert.Empty(t, s.CollectedData)
	assert.False(t, s.HandoffActive)
	assert.Nil(t, s.HandoffAt)
	assert.True(t, s.UpdatedAt.After(before))
}
