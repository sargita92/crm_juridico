package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestListUserTenantsUseCase_User_ReturnsLinkedTenants(t *testing.T) {
	userTenantRepo := newMockUserTenantRepo()
	tenantRepo := newMockTenantRepo()
	uc := NewListUserTenantsUseCase(userTenantRepo, tenantRepo)

	t1 := &tenantdomain.Tenant{ID: "t-1", Name: "A", Status: tenantdomain.TenantStatusActive}
	t2 := &tenantdomain.Tenant{ID: "t-2", Name: "B", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[t1.ID] = t1
	tenantRepo.tenants[t2.ID] = t2
	userTenantRepo.associations["user-1"] = []string{"t-1", "t-2"}

	tenants, err := uc.Execute(context.Background(), "user-1", domain.UserRoleUser)

	require.NoError(t, err)
	assert.Len(t, tenants, 2)
}

func TestListUserTenantsUseCase_Admin_ReturnsAllActiveTenants(t *testing.T) {
	userTenantRepo := newMockUserTenantRepo()
	tenantRepo := newMockTenantRepo()
	uc := NewListUserTenantsUseCase(userTenantRepo, tenantRepo)

	t1 := &tenantdomain.Tenant{ID: "t-1", Name: "A", Status: tenantdomain.TenantStatusActive}
	t2 := &tenantdomain.Tenant{ID: "t-2", Name: "B", Status: tenantdomain.TenantStatusBlocked}
	tenantRepo.tenants[t1.ID] = t1
	tenantRepo.tenants[t2.ID] = t2

	tenants, err := uc.Execute(context.Background(), "admin-1", domain.UserRoleAdmin)

	require.NoError(t, err)
	assert.Len(t, tenants, 1)
	assert.Equal(t, "t-1", tenants[0].ID)
}

func TestListUserTenantsUseCase_UserWithNoTenants_ReturnsNil(t *testing.T) {
	userTenantRepo := newMockUserTenantRepo()
	tenantRepo := newMockTenantRepo()
	uc := NewListUserTenantsUseCase(userTenantRepo, tenantRepo)

	tenants, err := uc.Execute(context.Background(), "user-orphan", domain.UserRoleUser)

	require.NoError(t, err)
	assert.Nil(t, tenants)
}
