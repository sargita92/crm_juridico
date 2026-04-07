package application

import (
	"context"
	"testing"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestActivateHandoff_Success(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	repo := &mockConvStateRepo{state: state}
	log := zap.NewNop()

	uc := NewActivateHandoffUseCase(repo, log)
	err := uc.Execute(context.Background(), "tenant-1", "conv-1")

	require.NoError(t, err)
	assert.True(t, repo.updated.HandoffActive)
	assert.NotNil(t, repo.updated.HandoffAt)
}

func TestActivateHandoff_AlreadyActive(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	state.ActivateHandoff()
	repo := &mockConvStateRepo{state: state}
	log := zap.NewNop()

	uc := NewActivateHandoffUseCase(repo, log)
	err := uc.Execute(context.Background(), "tenant-1", "conv-1")

	require.ErrorIs(t, err, ErrHandoffAlreadyActive)
}

func TestDeactivateHandoff_Success(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	state.ActivateHandoff()
	repo := &mockConvStateRepo{state: state}
	log := zap.NewNop()

	uc := NewDeactivateHandoffUseCase(repo, log)
	err := uc.Execute(context.Background(), "tenant-1", "conv-1")

	require.NoError(t, err)
	assert.False(t, repo.updated.HandoffActive)
	assert.NotNil(t, repo.updated.ResumedAt)
}

func TestDeactivateHandoff_NotActive(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	repo := &mockConvStateRepo{state: state}
	log := zap.NewNop()

	uc := NewDeactivateHandoffUseCase(repo, log)
	err := uc.Execute(context.Background(), "tenant-1", "conv-1")

	require.ErrorIs(t, err, ErrHandoffNotActive)
}
