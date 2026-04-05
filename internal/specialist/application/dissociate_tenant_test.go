package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestDissociateTenantUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")

	uc := NewDissociateTenantUseCase(specRepo, stRepo)
	err := uc.Execute(context.Background(), DissociateTenantInput{
		SpecialistID: "spec-1",
		TenantID:     "tenant-1",
	})

	require.NoError(t, err)

	exists, _ := stRepo.Exists(context.Background(), "spec-1", "tenant-1")
	assert.False(t, exists)
}

func TestDissociateTenantUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	uc := NewDissociateTenantUseCase(specRepo, stRepo)
	err := uc.Execute(context.Background(), DissociateTenantInput{
		SpecialistID: "nonexistent",
		TenantID:     "tenant-1",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestDissociateTenantUseCase_NotAssociated(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	s, _ := domain.NewSpecialist("spec-1", "Assistente", "desc", "prompt")
	specRepo.addSpecialist(s)

	uc := NewDissociateTenantUseCase(specRepo, stRepo)
	err := uc.Execute(context.Background(), DissociateTenantInput{
		SpecialistID: "spec-1",
		TenantID:     "tenant-1",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNotAssociated)
}
