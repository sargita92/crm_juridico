package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestDeactivateSpecialistUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	repo.addSpecialist(s)

	uc := NewDeactivateSpecialistUseCase(repo)
	err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, domain.SpecialistStatusInactive, repo.specialists["spec-1"].Status)
}

func TestDeactivateSpecialistUseCase_NotFound_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewDeactivateSpecialistUseCase(repo)

	err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestDeactivateSpecialistUseCase_AlreadyInactive_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	_ = s.Deactivate()
	repo.addSpecialist(s)

	uc := NewDeactivateSpecialistUseCase(repo)
	err := uc.Execute(context.Background(), "spec-1")

	assert.ErrorIs(t, err, domain.ErrSpecialistAlreadyInactive)
}
