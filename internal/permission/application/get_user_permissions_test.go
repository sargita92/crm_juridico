package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserPermissions_Success(t *testing.T) {
	p1 := newTestPerm("tenant-1", "", "user-1", "leads", "view")
	p2 := newTestPerm("tenant-1", "", "user-1", "funnels", "manage")
	p3 := newTestPerm("tenant-2", "", "user-1", "leads", "manage") // different tenant — excluded by repo
	repo := newMockPermissionRepo(p1, p2, p3)
	uc := NewManagePermissionsUseCase(repo)

	out, err := uc.GetUserPermissions(context.Background(), "tenant-1", "user-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
	resources := []string{out[0].Resource, out[1].Resource}
	assert.Contains(t, resources, "leads")
	assert.Contains(t, resources, "funnels")
}

func TestGetUserPermissions_Empty(t *testing.T) {
	repo := newMockPermissionRepo()
	uc := NewManagePermissionsUseCase(repo)

	out, err := uc.GetUserPermissions(context.Background(), "tenant-1", "user-1")

	require.NoError(t, err)
	assert.Len(t, out, 0)
}
