package application

import (
	"context"
	"testing"
	"time"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
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
	userRepo := newMockUserRepo()
	uc := NewManageMembersUseCase(groupRepo, ugRepo, userRepo)

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
	userRepo := newMockUserRepo()
	uc := NewManageMembersUseCase(groupRepo, ugRepo, userRepo)

	err := uc.AddMember(context.Background(), "user-1", "group-1", "tenant-1")

	assert.ErrorIs(t, err, domain.ErrUserAlreadyInGroup)
}

func TestManageMembers_RemoveMember_Success(t *testing.T) {
	groupRepo := newMockGroupRepo()
	ugRepo := newMockUserGroupRepo(newTestUserGroup("user-1", "group-1", "tenant-1"))
	userRepo := newMockUserRepo()
	uc := NewManageMembersUseCase(groupRepo, ugRepo, userRepo)

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
	userRepo := newMockUserRepo(
		&authdomain.User{ID: "user-1", Name: "User One", Email: "user1@example.com", CreatedAt: time.Now()},
		&authdomain.User{ID: "user-2", Name: "User Two", Email: "user2@example.com", CreatedAt: time.Now()},
		&authdomain.User{ID: "user-3", Name: "User Three", Email: "user3@example.com", CreatedAt: time.Now()},
	)
	uc := NewManageMembersUseCase(groupRepo, ugRepo, userRepo)

	members, err := uc.ListMembers(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, members, 2)
	assert.Equal(t, "user-1", members[0].UserID)
	assert.Equal(t, "User One", members[0].Name)
	assert.Equal(t, "user1@example.com", members[0].Email)
	assert.Equal(t, "user-2", members[1].UserID)
	assert.Equal(t, "User Two", members[1].Name)
	assert.Equal(t, "user2@example.com", members[1].Email)
}
