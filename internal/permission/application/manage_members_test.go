package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManageMembers_AddMember_Success(t *testing.T) {
	groupRepo := newMockGroupRepo(domain.PermissionGroup{
		ID:       "group-1",
		TenantID: "tenant-1",
		Name:     "Test Group",
	})
	ugRepo := newMockUserGroupRepo()
	uc := NewManageMembersUseCase(groupRepo, ugRepo)

	err := uc.AddMember(context.Background(), "user-1", "group-1", "tenant-1")

	require.NoError(t, err)
	assert.Len(t, ugRepo.userGroups, 1)
	assert.Equal(t, "user-1", ugRepo.userGroups[0].UserID)
	assert.Equal(t, "group-1", ugRepo.userGroups[0].GroupID)
}

func TestManageMembers_AddMember_AlreadyInGroup(t *testing.T) {
	groupRepo := newMockGroupRepo(domain.PermissionGroup{
		ID:       "group-1",
		TenantID: "tenant-1",
		Name:     "Test Group",
	})
	ugRepo := newMockUserGroupRepo(newTestUserGroup("user-1", "group-1", "tenant-1"))
	uc := NewManageMembersUseCase(groupRepo, ugRepo)

	err := uc.AddMember(context.Background(), "user-1", "group-1", "tenant-1")

	assert.ErrorIs(t, err, domain.ErrUserAlreadyInGroup)
}

func TestManageMembers_RemoveMember_Success(t *testing.T) {
	groupRepo := newMockGroupRepo()
	ugRepo := newMockUserGroupRepo(newTestUserGroup("user-1", "group-1", "tenant-1"))
	uc := NewManageMembersUseCase(groupRepo, ugRepo)

	err := uc.RemoveMember(context.Background(), "user-1", "group-1")

	require.NoError(t, err)
	assert.Len(t, ugRepo.userGroups, 0)
}

func TestManageMembers_ListMembers_Success(t *testing.T) {
	groupRepo := newMockGroupRepo()
	ugRepo := newMockUserGroupRepo(
		newTestUserGroup("user-1", "group-1", "tenant-1"),
		newTestUserGroup("user-2", "group-1", "tenant-1"),
		newTestUserGroup("user-3", "group-2", "tenant-1"),
	)
	uc := NewManageMembersUseCase(groupRepo, ugRepo)

	members, err := uc.ListMembers(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, "user-1", members[0].UserID)
	assert.Equal(t, "user-2", members[1].UserID)
}
