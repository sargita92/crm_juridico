package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateGroupUseCase_Success(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Old Name")
	repo := newMockGroupRepo(g)
	uc := NewUpdateGroupUseCase(repo)

	out, err := uc.Execute(context.Background(), UpdateGroupInput{
		TenantID:    "tenant-1",
		ID:          "group-1",
		Name:        "New Name",
		Description: "New description",
	})

	require.NoError(t, err)
	assert.Equal(t, "group-1", out.ID)
	assert.Equal(t, "New Name", out.Name)
	assert.Equal(t, "New description", out.Description)
}

func TestUpdateGroupUseCase_NotFound(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewUpdateGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		TenantID: "tenant-1",
		ID:       "nonexistent",
		Name:     "Name",
	})

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
}

func TestUpdateGroupUseCase_WrongTenant(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewUpdateGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		TenantID: "tenant-2",
		ID:       "group-1",
		Name:     "New Name",
	})

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
}

func TestUpdateGroupUseCase_EmptyName(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewUpdateGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		TenantID: "tenant-1",
		ID:       "group-1",
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrGroupNameRequired)
}
