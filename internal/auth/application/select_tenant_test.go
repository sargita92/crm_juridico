package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func setupSelectTenantUC() (*SelectTenantUseCase, *mockUserTenantRepo, *mockTenantRepo) {
	userTenantRepo := newMockUserTenantRepo()
	tenantRepo := newMockTenantRepo()
	tokenProvider := &mockTokenProvider{}

	uc := NewSelectTenantUseCase(userTenantRepo, tenantRepo, tokenProvider)
	return uc, userTenantRepo, tenantRepo
}

func TestSelectTenantUseCase_UserWithAccess_Success(t *testing.T) {
	uc, userTenantRepo, tenantRepo := setupSelectTenantUC()
	ctx := context.Background()

	tenant := &tenantdomain.Tenant{ID: "tenant-1", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[tenant.ID] = tenant
	userTenantRepo.associations["user-1"] = []string{"tenant-1"}

	output, err := uc.Execute(ctx, SelectTenantInput{
		UserID: "user-1", TenantID: "tenant-1", Role: domain.UserRoleUser,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.Token)
}

func TestSelectTenantUseCase_UserWithoutAccess_ReturnsError(t *testing.T) {
	uc, userTenantRepo, tenantRepo := setupSelectTenantUC()
	ctx := context.Background()

	tenant := &tenantdomain.Tenant{ID: "tenant-1", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[tenant.ID] = tenant
	userTenantRepo.associations["user-1"] = []string{"tenant-other"}

	_, err := uc.Execute(ctx, SelectTenantInput{
		UserID: "user-1", TenantID: "tenant-1", Role: domain.UserRoleUser,
	})

	assert.ErrorIs(t, err, ErrTenantNotAllowed)
}

func TestSelectTenantUseCase_AdminAccessAnyTenant(t *testing.T) {
	uc, _, tenantRepo := setupSelectTenantUC()
	ctx := context.Background()

	tenant := &tenantdomain.Tenant{ID: "tenant-1", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[tenant.ID] = tenant

	output, err := uc.Execute(ctx, SelectTenantInput{
		UserID: "admin-1", TenantID: "tenant-1", Role: domain.UserRoleAdmin,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.Token)
}

func TestSelectTenantUseCase_BlockedTenant_ReturnsError(t *testing.T) {
	uc, userTenantRepo, tenantRepo := setupSelectTenantUC()
	ctx := context.Background()

	tenant := &tenantdomain.Tenant{ID: "tenant-1", Status: tenantdomain.TenantStatusBlocked}
	tenantRepo.tenants[tenant.ID] = tenant
	userTenantRepo.associations["user-1"] = []string{"tenant-1"}

	_, err := uc.Execute(ctx, SelectTenantInput{
		UserID: "user-1", TenantID: "tenant-1", Role: domain.UserRoleUser,
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantBlocked)
}

func TestSelectTenantUseCase_TenantNotFound_ReturnsError(t *testing.T) {
	uc, _, _ := setupSelectTenantUC()

	_, err := uc.Execute(context.Background(), SelectTenantInput{
		UserID: "user-1", TenantID: "nonexistent", Role: domain.UserRoleUser,
	})

	assert.ErrorIs(t, err, tenantdomain.ErrTenantNotFound)
}
