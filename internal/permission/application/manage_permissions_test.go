package application

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	infra "github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagePermissions_SetGroupPermissions_Success(t *testing.T) {
	repo := newMockPermissionRepo()
	uc := NewManagePermissionsUseCase(repo)

	before := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("group", "updated"))
	err := uc.SetGroupPermissions(context.Background(), "tenant-1", "group-1", []PermissionInput{
		{Resource: "leads", Action: "view"},
		{Resource: "leads", Action: "manage"},
	})
	after := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("group", "updated"))

	require.NoError(t, err)
	assert.Len(t, repo.perms, 2)
	for _, p := range repo.perms {
		assert.Equal(t, "group-1", p.GroupID)
		assert.Equal(t, "leads", p.Resource)
	}
	assert.Equal(t, before+1, after)
}

func TestManagePermissions_SetGroupPermissions_InvalidResource(t *testing.T) {
	repo := newMockPermissionRepo()
	uc := NewManagePermissionsUseCase(repo)

	err := uc.SetGroupPermissions(context.Background(), "tenant-1", "group-1", []PermissionInput{
		{Resource: "nonexistent", Action: "view"},
	})

	assert.ErrorIs(t, err, domain.ErrInvalidResource)
	assert.Len(t, repo.perms, 0)
}

func TestManagePermissions_SetUserPermissions_Success(t *testing.T) {
	repo := newMockPermissionRepo()
	uc := NewManagePermissionsUseCase(repo)

	before := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("user", "updated"))
	err := uc.SetUserPermissions(context.Background(), "tenant-1", "user-1", []PermissionInput{
		{Resource: "funnels", Action: "manage"},
	})
	after := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("user", "updated"))

	require.NoError(t, err)
	assert.Len(t, repo.perms, 1)
	assert.Equal(t, "user-1", repo.perms[0].UserID)
	assert.Equal(t, "funnels", repo.perms[0].Resource)
	assert.Equal(t, before+1, after)
}

func TestManagePermissions_GetGroupPermissions_Success(t *testing.T) {
	repo := newMockPermissionRepo(
		newTestPerm("tenant-1", "group-1", "", "leads", "view"),
		newTestPerm("tenant-1", "group-1", "", "leads", "manage"),
		newTestPerm("tenant-1", "group-2", "", "leads", "view"),
	)
	uc := NewManagePermissionsUseCase(repo)

	out, err := uc.GetGroupPermissions(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
	for _, o := range out {
		assert.Equal(t, "leads", o.Resource)
	}
}

// --- F12 Step 7: permission.changed audit publishing ---

func ctxWithAdminClaims(userID, email string) context.Context {
	return middleware.SetClaimsForTest(context.Background(), &authdomain.TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   authdomain.UserRoleAdmin,
	})
}

func TestSetUserPermissions_AdminTarget_PublishesPermissionChanged(t *testing.T) {
	target := &authdomain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: authdomain.UserRoleAdmin, Status: authdomain.UserStatusActive,
		PasswordHash: "h",
	}
	permRepo := newMockPermissionRepo(
		newTestPerm("tenant-1", "", "u-1", "leads", "view"),
	)
	userRepo := newMockUserRepo(target)
	spy := &spyAuditPublisher{}

	uc := NewManagePermissionsUseCase(permRepo)
	uc.SetUserRepo(userRepo)
	uc.SetAuditPublisher(spy)

	ctx := ctxWithAdminClaims("admin-1", "admin@crm.com")
	err := uc.SetUserPermissions(ctx, "tenant-1", "u-1", []PermissionInput{
		{Resource: "leads", Action: "manage"},
		{Resource: "funnels", Action: "manage"},
	})
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionPermissionChanged, call.Action)
	assert.Equal(t, "user_admin", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "u-1", *call.EntityID)
	assert.Nil(t, call.TenantID, "MVP F12: alvo admin sempre tenant_id NULL")
	assert.Equal(t, "admin@crm.com", call.ActorEmail)

	// Diff em metadata: chave "permissions" com antes/depois ordenados
	require.NotNil(t, call.Metadata)
	permsDiff, ok := call.Metadata["permissions"].(map[string]any)
	require.True(t, ok, "diff de permissions presente")
	before := permsDiff["antes"].([]string)
	after := permsDiff["depois"].([]string)
	assert.Equal(t, []string{"leads:view"}, before)
	assert.Equal(t, []string{"funnels:manage", "leads:manage"}, after)
}

func TestSetUserPermissions_NonAdminTarget_DoesNotPublish(t *testing.T) {
	target := &authdomain.User{
		ID: "u-2", Name: "Carlos", Email: "carlos@crm.com",
		Role: authdomain.UserRoleUser, Status: authdomain.UserStatusActive,
		PasswordHash: "h",
	}
	permRepo := newMockPermissionRepo()
	userRepo := newMockUserRepo(target)
	spy := &spyAuditPublisher{}

	uc := NewManagePermissionsUseCase(permRepo)
	uc.SetUserRepo(userRepo)
	uc.SetAuditPublisher(spy)

	err := uc.SetUserPermissions(ctxWithAdminClaims("a", "a@x.com"), "tenant-1", "u-2", []PermissionInput{
		{Resource: "leads", Action: "view"},
	})
	require.NoError(t, err)
	assert.Empty(t, spy.calls, "alvo nao-admin NAO publica (S1-C15)")
}

func TestSetUserPermissions_NoChange_DoesNotPublish(t *testing.T) {
	target := &authdomain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: authdomain.UserRoleAdmin, Status: authdomain.UserStatusActive,
		PasswordHash: "h",
	}
	// before == after (mesma permissao)
	permRepo := newMockPermissionRepo(
		newTestPerm("tenant-1", "", "u-1", "leads", "view"),
	)
	userRepo := newMockUserRepo(target)
	spy := &spyAuditPublisher{}

	uc := NewManagePermissionsUseCase(permRepo)
	uc.SetUserRepo(userRepo)
	uc.SetAuditPublisher(spy)

	err := uc.SetUserPermissions(ctxWithAdminClaims("a", "a@x.com"), "tenant-1", "u-1", []PermissionInput{
		{Resource: "leads", Action: "view"},
	})
	require.NoError(t, err)
	assert.Empty(t, spy.calls, "diff vazio nao publica")
}

func TestSetUserPermissions_PublisherErrorDoesNotAbort(t *testing.T) {
	target := &authdomain.User{
		ID: "u-1", Name: "Ana", Email: "ana@crm.com",
		Role: authdomain.UserRoleAdmin, Status: authdomain.UserStatusActive,
		PasswordHash: "h",
	}
	permRepo := newMockPermissionRepo()
	userRepo := newMockUserRepo(target)
	spy := &spyAuditPublisher{publishErr: errors.New("audit failed")}

	uc := NewManagePermissionsUseCase(permRepo)
	uc.SetUserRepo(userRepo)
	uc.SetAuditPublisher(spy)

	err := uc.SetUserPermissions(ctxWithAdminClaims("a", "a@x.com"), "tenant-1", "u-1", []PermissionInput{
		{Resource: "leads", Action: "view"},
	})
	// S1-C17: auditoria falhando NAO aborta operacao
	require.NoError(t, err)
	assert.Len(t, permRepo.perms, 1, "permissao foi persistida apesar da falha de audit")
}

func TestSetUserPermissions_NoUserRepo_StillWorks(t *testing.T) {
	// UC sem userRepo (composicao legada): nao publica audit, mas opera normalmente.
	permRepo := newMockPermissionRepo()
	uc := NewManagePermissionsUseCase(permRepo)
	// sem SetUserRepo, sem SetAuditPublisher

	err := uc.SetUserPermissions(context.Background(), "tenant-1", "u-1", []PermissionInput{
		{Resource: "leads", Action: "view"},
	})
	require.NoError(t, err)
	assert.Len(t, permRepo.perms, 1)
}
