package application

import (
	"context"
	"testing"

	infra "github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sasrgita/crm-juridico/internal/permission/domain"
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
