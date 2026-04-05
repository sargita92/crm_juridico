package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestGetTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório ABC", domain.TenantTypePJ, "12345")
	repo.addTenant(tenant)

	uc := NewGetTenantUseCase(repo)

	output, err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)
	assert.Equal(t, "id-1", output.ID)
	assert.Equal(t, "Escritório ABC", output.Name)
	assert.Equal(t, "PJ", output.Type)
	assert.Equal(t, "12345", output.Document)
	assert.Equal(t, "active", output.Status)
}

func TestGetTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewGetTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestGetTenantUseCase_BlockedTenant_ShowsBlockReason(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "12345")
	_ = tenant.Block("Inadimplência")
	repo.addTenant(tenant)

	uc := NewGetTenantUseCase(repo)

	output, err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)
	assert.Equal(t, "blocked", output.Status)
	assert.Equal(t, "Inadimplência", output.BlockReason)
}
