package application

import (
	"context"
	"testing"
	"time"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGroup(id, tenantID, name string) domain.PermissionGroup {
	return domain.PermissionGroup{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestGetGroupUseCase_Success(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewGetGroupUseCase(repo)

	out, err := uc.Execute(context.Background(), "tenant-1", "group-1")

	require.NoError(t, err)
	assert.Equal(t, "group-1", out.ID)
	assert.Equal(t, "Admins", out.Name)
}

func TestGetGroupUseCase_NotFound(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewGetGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), "tenant-1", "nonexistent")

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
}

func TestGetGroupUseCase_WrongTenant(t *testing.T) {
	g := newTestGroup("group-1", "tenant-1", "Admins")
	repo := newMockGroupRepo(g)
	uc := NewGetGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), "tenant-2", "group-1")

	assert.ErrorIs(t, err, domain.ErrGroupNotFound)
}
