package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteGroupUseCase_Success(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewDeleteGroupUseCase(repo)

	err := uc.Execute(context.Background(), "tenant-1", "group-1")

	require.NoError(t, err)
	assert.Len(t, repo.groups, 0)
}

func TestDeleteGroupUseCase_NotFound(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewDeleteGroupUseCase(repo)

	err := uc.Execute(context.Background(), "tenant-1", "nonexistent")

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
}

func TestDeleteGroupUseCase_WrongTenant(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewDeleteGroupUseCase(repo)

	err := uc.Execute(context.Background(), "tenant-2", "group-1")

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
	assert.Len(t, repo.groups, 1, "group should not be deleted if tenant doesn't match")
}
