package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func setupManageUsersUC() (*ManageUsersUseCase, *mockUserRepo, *mockUserTenantRepo) {
	userRepo := newMockUserRepo()
	userTenantRepo := newMockUserTenantRepo()
	uc := NewManageUsersUseCase(userRepo, userTenantRepo)
	return uc, userRepo, userTenantRepo
}

func TestListTenantUsers_Success(t *testing.T) {
	uc, userRepo, userTenantRepo := setupManageUsersUC()
	ctx := context.Background()

	// Seed two users
	u1 := &domain.User{ID: "user-1", Name: "Alice", Email: "alice@email.com", Status: domain.UserStatusActive}
	u2 := &domain.User{ID: "user-2", Name: "Bob", Email: "bob@email.com", Status: domain.UserStatusActive}
	userRepo.users[u1.Email] = u1
	userRepo.users[u2.Email] = u2

	// Associate them to the tenant
	ut1 := &domain.UserTenant{UserID: "user-1", TenantID: "tenant-1", IsOwner: true, WhatsAppID: "+5511999990001"}
	ut2 := &domain.UserTenant{UserID: "user-2", TenantID: "tenant-1", IsOwner: false, WhatsAppID: ""}
	userTenantRepo.userTenants[utKey("user-1", "tenant-1")] = ut1
	userTenantRepo.userTenants[utKey("user-2", "tenant-1")] = ut2

	out, err := uc.ListTenantUsers(ctx, "tenant-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)

	outByID := make(map[string]UserOutput)
	for _, o := range out {
		outByID[o.ID] = o
	}

	assert.Equal(t, "Alice", outByID["user-1"].Name)
	assert.True(t, outByID["user-1"].IsOwner)
	assert.Equal(t, "+5511999990001", outByID["user-1"].WhatsAppID)
	assert.Equal(t, "Bob", outByID["user-2"].Name)
	assert.False(t, outByID["user-2"].IsOwner)
}

func TestListTenantUsers_EmptyTenant(t *testing.T) {
	uc, _, _ := setupManageUsersUC()
	ctx := context.Background()

	out, err := uc.ListTenantUsers(ctx, "tenant-empty")

	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestRemoveFromTenant_Success(t *testing.T) {
	uc, _, userTenantRepo := setupManageUsersUC()
	ctx := context.Background()

	userTenantRepo.userTenants[utKey("user-1", "tenant-1")] = &domain.UserTenant{
		UserID: "user-1", TenantID: "tenant-1", IsOwner: false,
	}
	userTenantRepo.owners[utKey("user-1", "tenant-1")] = false

	err := uc.RemoveFromTenant(ctx, "user-1", "tenant-1")

	require.NoError(t, err)
	assert.True(t, userTenantRepo.removed[utKey("user-1", "tenant-1")])
}

func TestRemoveFromTenant_CannotRemoveOwner(t *testing.T) {
	uc, _, userTenantRepo := setupManageUsersUC()
	ctx := context.Background()

	userTenantRepo.userTenants[utKey("owner-1", "tenant-1")] = &domain.UserTenant{
		UserID: "owner-1", TenantID: "tenant-1", IsOwner: true,
	}
	userTenantRepo.owners[utKey("owner-1", "tenant-1")] = true

	err := uc.RemoveFromTenant(ctx, "owner-1", "tenant-1")

	assert.ErrorIs(t, err, ErrCannotRemoveOwner)
	// Should NOT be in the removed map
	assert.False(t, userTenantRepo.removed[utKey("owner-1", "tenant-1")])
}

func TestSetWhatsAppID_Success(t *testing.T) {
	uc, _, userTenantRepo := setupManageUsersUC()
	ctx := context.Background()

	userTenantRepo.userTenants[utKey("user-1", "tenant-1")] = &domain.UserTenant{
		UserID: "user-1", TenantID: "tenant-1",
	}

	err := uc.SetWhatsAppID(ctx, "user-1", "tenant-1", "+5511988880000")

	require.NoError(t, err)
	assert.Equal(t, "+5511988880000", userTenantRepo.userTenants[utKey("user-1", "tenant-1")].WhatsAppID)
}
