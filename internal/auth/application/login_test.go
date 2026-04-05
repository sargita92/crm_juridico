package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func setupLoginUC() (*LoginUseCase, *mockUserRepo, *mockUserTenantRepo, *mockTenantRepo, *mockTokenProvider) {
	userRepo := newMockUserRepo()
	userTenantRepo := newMockUserTenantRepo()
	tenantRepo := newMockTenantRepo()
	hasher := &mockHasher{}
	tokenProvider := &mockTokenProvider{}

	uc := NewLoginUseCase(userRepo, userTenantRepo, tenantRepo, hasher, tokenProvider)
	return uc, userRepo, userTenantRepo, tenantRepo, tokenProvider
}

func TestLoginUseCase_ValidCredentials_OneTenant(t *testing.T) {
	uc, userRepo, userTenantRepo, tenantRepo, tokenProvider := setupLoginUC()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Name: "João", Email: "joao@email.com", PasswordHash: "hashed:secret123", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	userRepo.users[user.Email] = user

	tenant := &tenantdomain.Tenant{ID: "tenant-1", Name: "Escritório", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[tenant.ID] = tenant
	userTenantRepo.associations[user.ID] = []string{tenant.ID}

	output, err := uc.Execute(ctx, LoginInput{Email: "joao@email.com", Password: "secret123"})

	require.NoError(t, err)
	assert.NotEmpty(t, output.Token)
	assert.Equal(t, "user-1", output.UserID)
	assert.Equal(t, "João", output.UserName)
	assert.Len(t, output.Tenants, 1)
	// With 1 tenant, token should include tenant_id
	assert.Equal(t, "tenant-1", tokenProvider.lastClaims.TenantID)
}

func TestLoginUseCase_ValidCredentials_MultipleTenants(t *testing.T) {
	uc, userRepo, userTenantRepo, tenantRepo, tokenProvider := setupLoginUC()
	ctx := context.Background()

	user := &domain.User{ID: "user-1", Name: "João", Email: "joao@email.com", PasswordHash: "hashed:secret123", Role: domain.UserRoleUser, Status: domain.UserStatusActive}
	userRepo.users[user.Email] = user

	t1 := &tenantdomain.Tenant{ID: "tenant-1", Name: "Escritório A", Status: tenantdomain.TenantStatusActive}
	t2 := &tenantdomain.Tenant{ID: "tenant-2", Name: "Escritório B", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[t1.ID] = t1
	tenantRepo.tenants[t2.ID] = t2
	userTenantRepo.associations[user.ID] = []string{t1.ID, t2.ID}

	output, err := uc.Execute(ctx, LoginInput{Email: "joao@email.com", Password: "secret123"})

	require.NoError(t, err)
	assert.Len(t, output.Tenants, 2)
	// With multiple tenants, token should NOT include tenant_id
	assert.Empty(t, tokenProvider.lastClaims.TenantID)
}

func TestLoginUseCase_Admin_SeesAllTenants_NoAutoRedirect(t *testing.T) {
	uc, userRepo, _, tenantRepo, tokenProvider := setupLoginUC()
	ctx := context.Background()

	admin := &domain.User{ID: "admin-1", Name: "Admin", Email: "admin@email.com", PasswordHash: "hashed:admin123", Role: domain.UserRoleAdmin, Status: domain.UserStatusActive}
	userRepo.users[admin.Email] = admin

	// Admin is NOT associated to any tenant, but should see all
	t1 := &tenantdomain.Tenant{ID: "tenant-1", Name: "Escritório A", Status: tenantdomain.TenantStatusActive}
	t2 := &tenantdomain.Tenant{ID: "tenant-2", Name: "Escritório B", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[t1.ID] = t1
	tenantRepo.tenants[t2.ID] = t2

	output, err := uc.Execute(ctx, LoginInput{Email: "admin@email.com", Password: "admin123"})

	require.NoError(t, err)
	assert.Len(t, output.Tenants, 2)
	assert.Equal(t, domain.UserRoleAdmin, output.Role)
	// Admin never gets auto-redirect (no tenant_id in token)
	assert.Empty(t, tokenProvider.lastClaims.TenantID)
}

func TestLoginUseCase_Admin_OneTenant_StillNoAutoRedirect(t *testing.T) {
	uc, userRepo, _, tenantRepo, tokenProvider := setupLoginUC()
	ctx := context.Background()

	admin := &domain.User{ID: "admin-1", Name: "Admin", Email: "admin@email.com", PasswordHash: "hashed:admin123", Role: domain.UserRoleAdmin, Status: domain.UserStatusActive}
	userRepo.users[admin.Email] = admin

	t1 := &tenantdomain.Tenant{ID: "tenant-1", Name: "Escritório", Status: tenantdomain.TenantStatusActive}
	tenantRepo.tenants[t1.ID] = t1

	output, err := uc.Execute(ctx, LoginInput{Email: "admin@email.com", Password: "admin123"})

	require.NoError(t, err)
	assert.Len(t, output.Tenants, 1)
	// Even with 1 tenant, admin goes to select-tenant (no auto-redirect)
	assert.Empty(t, tokenProvider.lastClaims.TenantID)
}

func TestLoginUseCase_WrongPassword_ReturnsError(t *testing.T) {
	uc, userRepo, _, _, _ := setupLoginUC()

	user := &domain.User{ID: "user-1", Email: "joao@email.com", PasswordHash: "hashed:secret123", Status: domain.UserStatusActive}
	userRepo.users[user.Email] = user

	_, err := uc.Execute(context.Background(), LoginInput{Email: "joao@email.com", Password: "wrong"})
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginUseCase_NonexistentEmail_ReturnsError(t *testing.T) {
	uc, _, _, _, _ := setupLoginUC()

	_, err := uc.Execute(context.Background(), LoginInput{Email: "naoexiste@email.com", Password: "any"})
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLoginUseCase_InactiveUser_ReturnsError(t *testing.T) {
	uc, userRepo, _, _, _ := setupLoginUC()

	user := &domain.User{ID: "user-1", Email: "joao@email.com", PasswordHash: "hashed:secret123", Status: domain.UserStatusInactive}
	userRepo.users[user.Email] = user

	_, err := uc.Execute(context.Background(), LoginInput{Email: "joao@email.com", Password: "secret123"})
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
}
