package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestListSpecialistsUseCase_Success(t *testing.T) {
	repo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	s1, _ := domain.NewSpecialist("spec-1", "Assistente A", "desc", "prompt1")
	s2, _ := domain.NewSpecialist("spec-2", "Assistente B", "desc", "prompt2")
	repo.addSpecialist(s1)
	repo.addSpecialist(s2)

	// Associate spec-1 with 2 tenants
	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-1")
	_ = stRepo.Associate(context.Background(), "spec-1", "tenant-2")

	uc := NewListSpecialistsUseCase(repo, stRepo)
	output, err := uc.Execute(context.Background(), ListSpecialistsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(2), output.Total)
	assert.Len(t, output.Specialists, 2)

	// Find spec-1 and check tenant count
	for _, item := range output.Specialists {
		if item.ID == "spec-1" {
			assert.Equal(t, 2, item.TenantCount)
		}
		if item.ID == "spec-2" {
			assert.Equal(t, 0, item.TenantCount)
		}
	}
}

func TestListSpecialistsUseCase_FilterByStatus(t *testing.T) {
	repo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	active, _ := domain.NewSpecialist("spec-1", "Ativo", "", "prompt1")
	inactive, _ := domain.NewSpecialist("spec-2", "Inativo", "", "prompt2")
	_ = inactive.Deactivate()
	repo.addSpecialist(active)
	repo.addSpecialist(inactive)

	uc := NewListSpecialistsUseCase(repo, stRepo)
	output, err := uc.Execute(context.Background(), ListSpecialistsInput{Status: "active", Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Equal(t, "Ativo", output.Specialists[0].Name)
}

func TestListSpecialistsUseCase_EmptyResult(t *testing.T) {
	repo := newMockSpecialistRepo()
	stRepo := newMockSpecialistTenantRepo()

	uc := NewListSpecialistsUseCase(repo, stRepo)
	output, err := uc.Execute(context.Background(), ListSpecialistsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(0), output.Total)
	assert.Empty(t, output.Specialists)
}
