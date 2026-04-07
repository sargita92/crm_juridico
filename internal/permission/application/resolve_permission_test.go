package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tenantA = "tenant-a"
	userA   = "user-a"
	userB   = "user-b"
	groupA  = "group-a"
	groupB  = "group-b"
)

func buildUseCase(
	permRepo *mockPermissionRepo,
	ugRepo *mockUserGroupRepo,
	ownerChecker *mockOwnerChecker,
	adminChecker *mockAdminChecker,
) *ResolvePermissionUseCase {
	return NewResolvePermissionUseCase(permRepo, ugRepo, ownerChecker, adminChecker)
}

func TestResolvePermission_OwnerBypass(t *testing.T) {
	permRepo := newMockPermissionRepo() // no permissions configured
	ugRepo := newMockUserGroupRepo()    // no group memberships
	ownerChecker := newMockOwnerChecker()
	ownerChecker.setOwner(userA, tenantA)
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	ok, err := uc.HasPermission(context.Background(), userA, tenantA, "leads", "manage")
	require.NoError(t, err)
	assert.True(t, ok, "owner should always have permission")
}

func TestResolvePermission_AdminBypass(t *testing.T) {
	permRepo := newMockPermissionRepo()
	ugRepo := newMockUserGroupRepo()
	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()
	adminChecker.setAdmin(userA)

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	ok, err := uc.HasPermission(context.Background(), userA, tenantA, "settings", "manage")
	require.NoError(t, err)
	assert.True(t, ok, "admin should always have permission")
}

func TestResolvePermission_IndividualPermission(t *testing.T) {
	perm := newTestPerm(tenantA, "", userA, "leads", "view")
	permRepo := newMockPermissionRepo(perm)
	ugRepo := newMockUserGroupRepo()
	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	ok, err := uc.HasPermission(context.Background(), userA, tenantA, "leads", "view")
	require.NoError(t, err)
	assert.True(t, ok, "user with individual permission should have access")
}

func TestResolvePermission_GroupPermission(t *testing.T) {
	// userA is in groupA, groupA has leads:manage
	perm := newTestPerm(tenantA, groupA, "", "leads", "manage")
	permRepo := newMockPermissionRepo(perm)

	ug := newTestUserGroup(userA, groupA, tenantA)
	ugRepo := newMockUserGroupRepo(ug)

	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	ok, err := uc.HasPermission(context.Background(), userA, tenantA, "leads", "manage")
	require.NoError(t, err)
	assert.True(t, ok, "user in group with permission should have access")
}

func TestResolvePermission_UnionAcrossGroups(t *testing.T) {
	// userA is in groupA (leads:view) and groupB (funnels:manage)
	permA := newTestPerm(tenantA, groupA, "", "leads", "view")
	permB := newTestPerm(tenantA, groupB, "", "funnels", "manage")
	permRepo := newMockPermissionRepo(permA, permB)

	ugA := newTestUserGroup(userA, groupA, tenantA)
	ugB := newTestUserGroup(userA, groupB, tenantA)
	ugRepo := newMockUserGroupRepo(ugA, ugB)

	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	// Both permissions from both groups should be accessible
	okLeads, err := uc.HasPermission(context.Background(), userA, tenantA, "leads", "view")
	require.NoError(t, err)
	assert.True(t, okLeads, "user should have leads:view from groupA")

	okFunnels, err := uc.HasPermission(context.Background(), userA, tenantA, "funnels", "manage")
	require.NoError(t, err)
	assert.True(t, okFunnels, "user should have funnels:manage from groupB")
}

func TestResolvePermission_NoPermission(t *testing.T) {
	permRepo := newMockPermissionRepo()
	ugRepo := newMockUserGroupRepo()
	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	ok, err := uc.HasPermission(context.Background(), userB, tenantA, "leads", "manage")
	require.NoError(t, err)
	assert.False(t, ok, "user with no permissions should be denied")
}

func TestResolvePermission_IndividualOverridesEmptyGroup(t *testing.T) {
	// userA is in groupA which has NO permissions, but has individual leads:view
	individualPerm := newTestPerm(tenantA, "", userA, "leads", "view")
	permRepo := newMockPermissionRepo(individualPerm)

	ug := newTestUserGroup(userA, groupA, tenantA)
	ugRepo := newMockUserGroupRepo(ug)

	ownerChecker := newMockOwnerChecker()
	adminChecker := newMockAdminChecker()

	uc := buildUseCase(permRepo, ugRepo, ownerChecker, adminChecker)

	// Individual perm should still grant access even though the group has none
	ok, err := uc.HasPermission(context.Background(), userA, tenantA, "leads", "view")
	require.NoError(t, err)
	assert.True(t, ok, "individual permission should grant access even when group has none")

	// A resource the user has neither individual nor group permission for
	okDenied, err := uc.HasPermission(context.Background(), userA, tenantA, "settings", "manage")
	require.NoError(t, err)
	assert.False(t, okDenied, "user should be denied for resources with no permission")
}
