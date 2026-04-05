package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestListTenantsUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant1, _ := domain.NewTenant("id-1", "Escritório A", domain.TenantTypePJ, "111")
	tenant2, _ := domain.NewTenant("id-2", "Dr. João", domain.TenantTypePF, "222")
	repo.addTenant(tenant1)
	repo.addTenant(tenant2)

	uc := NewListTenantsUseCase(repo)

	output, err := uc.Execute(context.Background(), ListTenantsInput{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), output.Total)
	assert.Len(t, output.Tenants, 2)
}

func TestListTenantsUseCase_FilterByStatus(t *testing.T) {
	repo := newMockTenantRepo()
	active, _ := domain.NewTenant("id-1", "Ativo", domain.TenantTypePJ, "111")
	blocked, _ := domain.NewTenant("id-2", "Bloqueado", domain.TenantTypePJ, "222")
	_ = blocked.Block("Motivo")
	repo.addTenant(active)
	repo.addTenant(blocked)

	uc := NewListTenantsUseCase(repo)

	output, err := uc.Execute(context.Background(), ListTenantsInput{Status: "blocked", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Equal(t, "Bloqueado", output.Tenants[0].Name)
}

func TestListTenantsUseCase_FilterByType(t *testing.T) {
	repo := newMockTenantRepo()
	pj, _ := domain.NewTenant("id-1", "PJ", domain.TenantTypePJ, "111")
	pf, _ := domain.NewTenant("id-2", "PF", domain.TenantTypePF, "222")
	repo.addTenant(pj)
	repo.addTenant(pf)

	uc := NewListTenantsUseCase(repo)

	output, err := uc.Execute(context.Background(), ListTenantsInput{Type: "PF", Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Equal(t, "PF", output.Tenants[0].Name)
}

func TestListTenantsUseCase_EmptyResult(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewListTenantsUseCase(repo)

	output, err := uc.Execute(context.Background(), ListTenantsInput{Page: 1, Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(0), output.Total)
	assert.Empty(t, output.Tenants)
}
