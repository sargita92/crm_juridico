package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpecialist_WithValidData_ReturnsSpecialist(t *testing.T) {
	s, err := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads de direito civil", "Voce e um assistente...")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", s.ID)
	assert.Equal(t, "Assistente Juridico", s.Name)
	assert.Equal(t, "Atende leads de direito civil", s.Description)
	assert.Equal(t, "Voce e um assistente...", s.Prompt)
	assert.Equal(t, SpecialistStatusActive, s.Status)
	assert.False(t, s.CreatedAt.IsZero())
	assert.False(t, s.UpdatedAt.IsZero())
}

func TestNewSpecialist_WithoutDescription_ReturnsSpecialist(t *testing.T) {
	s, err := NewSpecialist("uuid-1", "Assistente", "", "Prompt valido")

	require.NoError(t, err)
	assert.Empty(t, s.Description)
}

func TestNewSpecialist_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewSpecialist("uuid-1", "", "desc", "prompt")

	assert.ErrorIs(t, err, ErrSpecialistNameRequired)
}

func TestNewSpecialist_EmptyPrompt_ReturnsError(t *testing.T) {
	_, err := NewSpecialist("uuid-1", "Nome", "desc", "")

	assert.ErrorIs(t, err, ErrSpecialistPromptRequired)
}

func TestNewSpecialist_PromptTooLong_ReturnsError(t *testing.T) {
	longPrompt := make([]byte, MaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	_, err := NewSpecialist("uuid-1", "Nome", "desc", string(longPrompt))

	assert.ErrorIs(t, err, ErrPromptTooLong)
}

func TestNewSpecialist_DescriptionTooLong_ReturnsError(t *testing.T) {
	longDesc := make([]byte, MaxDescriptionLength+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	_, err := NewSpecialist("uuid-1", "Nome", string(longDesc), "prompt")

	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

func TestSpecialist_IsActive(t *testing.T) {
	s := &Specialist{Status: SpecialistStatusActive}
	assert.True(t, s.IsActive())

	s.Status = SpecialistStatusInactive
	assert.False(t, s.IsActive())
}

func TestSpecialist_Update_Success(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Nome Antigo", "desc antiga", "prompt antigo")

	err := s.Update("Nome Novo", "desc nova", "prompt novo")

	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", s.Name)
	assert.Equal(t, "desc nova", s.Description)
	assert.Equal(t, "prompt novo", s.Prompt)
}

func TestSpecialist_Update_EmptyName_ReturnsError(t *testing.T) {
	s := &Specialist{Name: "Antigo"}

	err := s.Update("", "desc", "prompt")

	assert.ErrorIs(t, err, ErrSpecialistNameRequired)
	assert.Equal(t, "Antigo", s.Name)
}

func TestSpecialist_Update_EmptyPrompt_ReturnsError(t *testing.T) {
	s := &Specialist{Prompt: "antigo"}

	err := s.Update("Nome", "desc", "")

	assert.ErrorIs(t, err, ErrSpecialistPromptRequired)
	assert.Equal(t, "antigo", s.Prompt)
}

func TestSpecialist_Update_PromptTooLong_ReturnsError(t *testing.T) {
	s := &Specialist{Prompt: "antigo"}
	longPrompt := make([]byte, MaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	err := s.Update("Nome", "desc", string(longPrompt))

	assert.ErrorIs(t, err, ErrPromptTooLong)
}

func TestSpecialist_Update_DescriptionTooLong_ReturnsError(t *testing.T) {
	s := &Specialist{Description: "antiga"}
	longDesc := make([]byte, MaxDescriptionLength+1)
	for i := range longDesc {
		longDesc[i] = 'a'
	}

	err := s.Update("Nome", string(longDesc), "prompt")

	assert.ErrorIs(t, err, ErrDescriptionTooLong)
}

func TestSpecialist_UpdatePrompt_Success(t *testing.T) {
	s := &Specialist{Prompt: "antigo"}

	err := s.UpdatePrompt("novo prompt")

	require.NoError(t, err)
	assert.Equal(t, "novo prompt", s.Prompt)
}

func TestSpecialist_UpdatePrompt_Empty_ReturnsError(t *testing.T) {
	s := &Specialist{Prompt: "antigo"}

	err := s.UpdatePrompt("")

	assert.ErrorIs(t, err, ErrSpecialistPromptRequired)
	assert.Equal(t, "antigo", s.Prompt)
}

func TestSpecialist_UpdatePrompt_TooLong_ReturnsError(t *testing.T) {
	s := &Specialist{Prompt: "antigo"}
	longPrompt := make([]byte, MaxPromptLength+1)
	for i := range longPrompt {
		longPrompt[i] = 'a'
	}

	err := s.UpdatePrompt(string(longPrompt))

	assert.ErrorIs(t, err, ErrPromptTooLong)
	assert.Equal(t, "antigo", s.Prompt)
}

func TestSpecialist_Deactivate_Success(t *testing.T) {
	s := &Specialist{Status: SpecialistStatusActive}

	err := s.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, SpecialistStatusInactive, s.Status)
}

func TestSpecialist_Deactivate_AlreadyInactive_ReturnsError(t *testing.T) {
	s := &Specialist{Status: SpecialistStatusInactive}

	err := s.Deactivate()

	assert.ErrorIs(t, err, ErrSpecialistAlreadyInactive)
}

func TestSpecialist_Activate_Success(t *testing.T) {
	s := &Specialist{Status: SpecialistStatusInactive}

	err := s.Activate()

	require.NoError(t, err)
	assert.Equal(t, SpecialistStatusActive, s.Status)
}

func TestSpecialist_Activate_AlreadyActive_ReturnsError(t *testing.T) {
	s := &Specialist{Status: SpecialistStatusActive}

	err := s.Activate()

	assert.ErrorIs(t, err, ErrSpecialistAlreadyActive)
}

func TestSpecialist_CrossSellDefaultsDisabled(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads", "Voce e um assistente...")

	assert.False(t, s.CrossSellEnabled)
	assert.Equal(t, CrossSellModeAnnounce, s.CrossSellMode)
}

func TestSpecialist_EnableCrossSell_RequiresTemplateInAnnounceMode(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads", "Voce e um assistente...")

	err := s.EnableCrossSell(CrossSellModeAnnounce, "")

	require.ErrorIs(t, err, ErrCrossSellTemplateRequired)
}

func TestSpecialist_EnableCrossSell_SilentDoesNotRequireTemplate(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads", "Voce e um assistente...")

	err := s.EnableCrossSell(CrossSellModeSilent, "")

	require.NoError(t, err)
	assert.True(t, s.CrossSellEnabled)
	assert.Equal(t, CrossSellModeSilent, s.CrossSellMode)
}

func TestSpecialist_EnableCrossSell_AnnounceWithTemplate_Success(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads", "Voce e um assistente...")

	err := s.EnableCrossSell(CrossSellModeAnnounce, "Conheça nossos outros serviços")

	require.NoError(t, err)
	assert.True(t, s.CrossSellEnabled)
	assert.Equal(t, CrossSellModeAnnounce, s.CrossSellMode)
	assert.Equal(t, "Conheça nossos outros serviços", s.CrossSellAnnouncementTemplate)
}

func TestSpecialist_DisableCrossSell_Success(t *testing.T) {
	s, _ := NewSpecialist("uuid-1", "Assistente Juridico", "Atende leads", "Voce e um assistente...")
	_ = s.EnableCrossSell(CrossSellModeAnnounce, "Template")

	s.DisableCrossSell()

	assert.False(t, s.CrossSellEnabled)
}
