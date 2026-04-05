package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestActivateSpecialistUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	_ = s.Deactivate()
	repo.addSpecialist(s)

	uc := NewActivateSpecialistUseCase(repo)
	err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, domain.SpecialistStatusActive, repo.specialists["spec-1"].Status)
}

func TestActivateSpecialistUseCase_NotFound_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewActivateSpecialistUseCase(repo)

	err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestActivateSpecialistUseCase_AlreadyActive_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	repo.addSpecialist(s)

	uc := NewActivateSpecialistUseCase(repo)
	err := uc.Execute(context.Background(), "spec-1")

	assert.ErrorIs(t, err, domain.ErrSpecialistAlreadyActive)
}
