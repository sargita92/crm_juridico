package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestGetSpecialistUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	repo.addSpecialist(s)

	uc := NewGetSpecialistUseCase(repo)
	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, "spec-1", output.ID)
	assert.Equal(t, "Assistente", output.Name)
	assert.Equal(t, "desc", output.Description)
	assert.Equal(t, "prompt", output.Prompt)
	assert.Equal(t, "active", output.Status)
}

func TestGetSpecialistUseCase_NotFound_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewGetSpecialistUseCase(repo)

	_, err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}
