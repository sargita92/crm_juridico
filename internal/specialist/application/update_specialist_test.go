package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestUpdateSpecialistUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Nome Antigo", "desc antiga", "prompt antigo")
	repo.addSpecialist(s)

	uc := NewUpdateSpecialistUseCase(repo)
	output, err := uc.Execute(context.Background(), UpdateSpecialistInput{
		ID:          "spec-1",
		Name:        "Nome Novo",
		Description: "desc nova",
		Prompt:      "prompt novo",
	})

	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", output.Name)
	assert.Equal(t, "desc nova", output.Description)
	assert.Equal(t, "prompt novo", output.Prompt)
}

func TestUpdateSpecialistUseCase_NotFound_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewUpdateSpecialistUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateSpecialistInput{
		ID:     "nonexistent",
		Name:   "Nome",
		Prompt: "prompt",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestUpdateSpecialistUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Nome", "desc", "prompt")
	repo.addSpecialist(s)

	uc := NewUpdateSpecialistUseCase(repo)
	_, err := uc.Execute(context.Background(), UpdateSpecialistInput{
		ID:     "spec-1",
		Name:   "",
		Prompt: "prompt",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNameRequired)
}

func TestUpdateSpecialistUseCase_EmptyPrompt_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Nome", "desc", "prompt")
	repo.addSpecialist(s)

	uc := NewUpdateSpecialistUseCase(repo)
	_, err := uc.Execute(context.Background(), UpdateSpecialistInput{
		ID:     "spec-1",
		Name:   "Nome",
		Prompt: "",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistPromptRequired)
}

func TestUpdateSpecialistUseCase_PromptTooLong_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	s, _ := domain.NewSpecialist("spec-1", "Nome", "desc", "prompt")
	repo.addSpecialist(s)

	longPrompt := make([]byte, domain.MaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	uc := NewUpdateSpecialistUseCase(repo)
	_, err := uc.Execute(context.Background(), UpdateSpecialistInput{
		ID:     "spec-1",
		Name:   "Nome",
		Prompt: string(longPrompt),
	})

	assert.ErrorIs(t, err, domain.ErrPromptTooLong)
}
