package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
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

// --- Admin User UCs (F12 Step 7) ---

// ctxWithAdminClaims monta um context com TokenClaims (admin) usado pelos
// UCs para popular ActorEmail e UserID no audit publisher.
func ctxWithAdminClaims(userID, email string) context.Context {
	return middleware.SetClaimsForTest(context.Background(), &domain.TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   domain.UserRoleAdmin,
	})
}

func TestCreateAdminUser_Success_PublishesAudit(t *testing.T) {
	uc, _, _ := setupManageUsersUC()
	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	ctx := ctxWithAdminClaims("admin-1", "admin@crm.com")

	out, err := uc.CreateAdminUser(ctx, CreateAdminUserInput{
		Name:         "Ana",
		Email:        "ana@crm.com",
		PasswordHash: "h:secret",
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, domain.UserRoleAdmin, out.Role)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionUserAdminCreated, call.Action)
	assert.Equal(t, "user_admin", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, out.ID, *call.EntityID)
	assert.Nil(t, call.TenantID, "user_admin actions sem tenant alvo")
	assert.Equal(t, "admin@crm.com", call.ActorEmail)
	require.NotNil(t, call.UserID)
	assert.Equal(t, "admin-1", *call.UserID)
	// Metadata captura name + email para diagnostico humano
	assert.Equal(t, "Ana", call.Metadata["name"])
	assert.Equal(t, "ana@crm.com", call.Metadata["email"])
}

func TestCreateAdminUser_DoesNotPublishWhenRepoFails(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	userRepo.createErr = errors.New("db down")
	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	_, err := uc.CreateAdminUser(ctxWithAdminClaims("a", "a@x.com"), CreateAdminUserInput{
		Name: "Ana", Email: "ana@crm.com", PasswordHash: "h",
	})
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestCreateAdminUser_PublisherErrorDoesNotAbort(t *testing.T) {
	uc, _, _ := setupManageUsersUC()
	spy := &spyAuditPublisher{publishErr: errors.New("audit failed")}
	uc.SetAuditPublisher(spy)

	out, err := uc.CreateAdminUser(ctxWithAdminClaims("a", "a@x.com"), CreateAdminUserInput{
		Name: "Ana", Email: "ana@crm.com", PasswordHash: "h",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestCreateAdminUser_NilPublisher_NoPanic(t *testing.T) {
	uc, _, _ := setupManageUsersUC()
	// Nao chama SetAuditPublisher — deve usar NoopPublisher default

	out, err := uc.CreateAdminUser(ctxWithAdminClaims("a", "a@x.com"), CreateAdminUserInput{
		Name: "Ana", Email: "ana@crm.com", PasswordHash: "h",
	})
	require.NoError(t, err)
	require.NotNil(t, out)

	// SetAuditPublisher(nil) deve cair pra Noop sem panic
	uc.SetAuditPublisher(nil)
	out2, err := uc.CreateAdminUser(ctxWithAdminClaims("a", "a@x.com"), CreateAdminUserInput{
		Name: "Bob", Email: "bob@crm.com", PasswordHash: "h",
	})
	require.NoError(t, err)
	require.NotNil(t, out2)
}

func TestUpdateAdminUser_Success_PublishesDiff(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	ctx := ctxWithAdminClaims("admin-1", "admin@crm.com")

	err := uc.UpdateAdminUser(ctx, UpdateAdminUserInput{
		ID:    "u-1",
		Name:  "Ana Maria",
		Email: "ana.maria@crm.com",
	})
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionUserAdminUpdated, call.Action)
	assert.Equal(t, "user_admin", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "u-1", *call.EntityID)
	assert.Nil(t, call.TenantID)

	// Diff em metadata
	require.NotNil(t, call.Metadata)
	nameDiff, ok := call.Metadata["name"].(map[string]any)
	require.True(t, ok, "diff de name presente")
	assert.Equal(t, "Ana", nameDiff["antes"])
	assert.Equal(t, "Ana Maria", nameDiff["depois"])

	emailDiff, ok := call.Metadata["email"].(map[string]any)
	require.True(t, ok, "diff de email presente")
	assert.Equal(t, "ana@crm.com", emailDiff["antes"])
	assert.Equal(t, "ana.maria@crm.com", emailDiff["depois"])
}

func TestUpdateAdminUser_NoChanges_DoesNotPublish(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	ctx := ctxWithAdminClaims("admin-1", "admin@crm.com")

	err := uc.UpdateAdminUser(ctx, UpdateAdminUserInput{
		ID:    "u-1",
		Name:  "Ana",
		Email: "ana@crm.com",
	})
	require.NoError(t, err)
	assert.Empty(t, spy.calls, "update sem mudancas nao publica")
}

func TestUpdateAdminUser_NonAdminTarget_DoesNotPublish(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-2", Name: "Carlos", Email: "carlos@crm.com",
		Role: domain.UserRoleUser, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.UpdateAdminUser(ctxWithAdminClaims("a", "a@x.com"), UpdateAdminUserInput{
		ID:    "u-2",
		Name:  "Carlos S.",
		Email: "carlos@crm.com",
	})
	require.NoError(t, err)
	assert.Empty(t, spy.calls, "alvo nao-admin nao publica em ManageUsers")
}

func TestUpdateAdminUser_RepoFails_DoesNotPublish(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing
	userRepo.updateErr = errors.New("db down")

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.UpdateAdminUser(ctxWithAdminClaims("a", "a@x.com"), UpdateAdminUserInput{
		ID:    "u-1",
		Name:  "Ana Maria",
		Email: "ana@crm.com",
	})
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestDeactivateAdminUser_Success_PublishesAudit(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.DeactivateAdminUser(ctxWithAdminClaims("a", "a@x.com"), "u-1")
	require.NoError(t, err)
	assert.Equal(t, domain.UserStatusInactive, userRepo.users[existing.Email].Status)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionUserAdminDeactivated, call.Action)
	assert.Equal(t, "user_admin", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "u-1", *call.EntityID)
	assert.Nil(t, call.TenantID)
}

func TestDeactivateAdminUser_NonAdminTarget_DoesNotPublish(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-2", Name: "Carlos", Email: "carlos@crm.com",
		Role: domain.UserRoleUser, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.DeactivateAdminUser(ctxWithAdminClaims("a", "a@x.com"), "u-2")
	require.NoError(t, err)
	assert.Empty(t, spy.calls)
}

func TestBlockAdminUser_Success_PublishesAuditWithReason(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.BlockAdminUser(ctxWithAdminClaims("a", "a@x.com"), "u-1", "violacao de politica")
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionUserAdminBlocked, call.Action)
	assert.Equal(t, "violacao de politica", call.Metadata["motivo"])
}

func TestUnblockAdminUser_Success_PublishesAudit(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: domain.UserRoleAdmin, Status: domain.UserStatusInactive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.UnblockAdminUser(ctxWithAdminClaims("a", "a@x.com"), "u-1", "esclarecido")
	require.NoError(t, err)
	assert.Equal(t, domain.UserStatusActive, userRepo.users[existing.Email].Status)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionUserAdminUnblocked, call.Action)
	assert.Equal(t, "esclarecido", call.Metadata["motivo"])
}

func TestBlockAdminUser_NonAdminTarget_DoesNotPublish(t *testing.T) {
	uc, userRepo, _ := setupManageUsersUC()
	existing := &domain.User{
		ID: "u-2", Name: "Carlos", Email: "carlos@crm.com",
		Role: domain.UserRoleUser, Status: domain.UserStatusActive,
		PasswordHash: "h",
	}
	userRepo.users[existing.Email] = existing

	spy := &spyAuditPublisher{}
	uc.SetAuditPublisher(spy)

	err := uc.BlockAdminUser(ctxWithAdminClaims("a", "a@x.com"), "u-2", "qualquer")
	require.NoError(t, err)
	assert.Empty(t, spy.calls)
}
