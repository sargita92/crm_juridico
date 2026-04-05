package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestUpdateTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome Antigo", domain.TenantTypePF, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	output, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-1",
		Name:     "Nome Novo",
		Type:     "PJ",
		Document: "222",
	})

	require.NoError(t, err)
	assert.Equal(t, "Nome Novo", output.Name)
	assert.Equal(t, "PJ", output.Type)
	assert.Equal(t, "222", output.Document)
}

func TestUpdateTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "nonexistent",
		Name:     "Nome",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestUpdateTenantUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Nome", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-1",
		Name:     "",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNameRequired)
}

func TestUpdateTenantUseCase_DuplicateDocument_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	t1, _ := domain.NewTenant("id-1", "A", domain.TenantTypePJ, "111")
	t2, _ := domain.NewTenant("id-2", "B", domain.TenantTypePJ, "222")
	repo.addTenant(t1)
	repo.addTenant(t2)

	uc := NewUpdateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateTenantInput{
		ID:       "id-2",
		Name:     "B",
		Type:     "PJ",
		Document: "111",
	})

	assert.ErrorIs(t, err, domain.ErrTenantDocumentExists)
}
