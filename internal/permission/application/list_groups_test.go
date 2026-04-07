package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListGroupsUseCase_Success(t *testing.T) {
	g1 := newTestGroup("group-1", "tenant-1", "Admins")
	g2 := newTestGroup("group-2", "tenant-1", "Operators")
	g3 := newTestGroup("group-3", "tenant-2", "Other Tenant Group")
	repo := newMockGroupRepo(g1, g2, g3)
	uc := NewListGroupsUseCase(repo)

	out, err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
	ids := []string{out[0].ID, out[1].ID}
	assert.Contains(t, ids, "group-1")
	assert.Contains(t, ids, "group-2")
}

func TestListGroupsUseCase_Empty(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewListGroupsUseCase(repo)

	out, err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	assert.Len(t, out, 0)
}
