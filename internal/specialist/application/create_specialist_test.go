package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestCreateSpecialistUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewCreateSpecialistUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateSpecialistInput{
		Name:        "Assistente Juridico",
		Description: "Atende leads de direito civil",
		Prompt:      "Voce e um assistente juridico...",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Assistente Juridico", output.Name)
	assert.Equal(t, "Atende leads de direito civil", output.Description)
	assert.Equal(t, "active", output.Status)
}

func TestCreateSpecialistUseCase_WithoutDescription_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewCreateSpecialistUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateSpecialistInput{
		Name:   "Assistente",
		Prompt: "Prompt valido",
	})

	require.NoError(t, err)
	assert.Empty(t, output.Description)
}

func TestCreateSpecialistUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewCreateSpecialistUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateSpecialistInput{
		Name:   "",
		Prompt: "Prompt valido",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNameRequired)
}

func TestCreateSpecialistUseCase_EmptyPrompt_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewCreateSpecialistUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateSpecialistInput{
		Name:   "Assistente",
		Prompt: "",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistPromptRequired)
}

func TestCreateSpecialistUseCase_PromptTooLong_ReturnsError(t *testing.T) {
	repo := newMockSpecialistRepo()
	uc := NewCreateSpecialistUseCase(repo)

	longPrompt := make([]byte, domain.MaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	_, err := uc.Execute(context.Background(), CreateSpecialistInput{
		Name:   "Assistente",
		Prompt: string(longPrompt),
	})

	assert.ErrorIs(t, err, domain.ErrPromptTooLong)
}
