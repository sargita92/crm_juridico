package application

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	infra "github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGroupFunnel(id, groupID, funnelID string, columnIDs []string) domain.GroupFunnel {
	return domain.GroupFunnel{
		ID:        id,
		GroupID:   groupID,
		FunnelID:  funnelID,
		ColumnIDs: columnIDs,
		CreatedAt: time.Now(),
	}
}

func TestManageGroupFunnels_SetGroupFunnel_Success(t *testing.T) {
	repo := newMockGroupFunnelRepo()
	uc := NewManageGroupFunnelsUseCase(repo)

	before := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("funnel", "updated"))
	err := uc.SetGroupFunnel(context.Background(), GroupFunnelInput{
		GroupID:   "group-1",
		FunnelID:  "funnel-1",
		ColumnIDs: []string{"col-1", "col-2"},
	})
	after := testutil.ToFloat64(infra.ChangesTotal.WithLabelValues("funnel", "updated"))

	require.NoError(t, err)
	assert.Len(t, repo.funnels, 1)
	assert.Equal(t, "group-1", repo.funnels[0].GroupID)
	assert.Equal(t, "funnel-1", repo.funnels[0].FunnelID)
	assert.Equal(t, []string{"col-1", "col-2"}, repo.funnels[0].ColumnIDs)
	assert.Equal(t, before+1, after)
}

func TestManageGroupFunnels_SetGroupFunnel_Update(t *testing.T) {
	existing := newTestGroupFunnel("gf-1", "group-1", "funnel-1", []string{"col-1"})
	repo := newMockGroupFunnelRepo(existing)
	uc := NewManageGroupFunnelsUseCase(repo)

	err := uc.SetGroupFunnel(context.Background(), GroupFunnelInput{
		GroupID:   "group-1",
		FunnelID:  "funnel-1",
		ColumnIDs: []string{"col-1", "col-2"},
	})

	require.NoError(t, err)
	assert.Len(t, repo.funnels, 1, "should upsert, not append")
}

func TestManageGroupFunnels_SetGroupFunnel_MissingGroupID(t *testing.T) {
	repo := newMockGroupFunnelRepo()
	uc := NewManageGroupFunnelsUseCase(repo)

	err := uc.SetGroupFunnel(context.Background(), GroupFunnelInput{
		GroupID:  "",
		FunnelID: "funnel-1",
	})

	assert.ErrorIs(t, err, domain.ErrGroupIDRequired)
}

func TestManageGroupFunnels_SetGroupFunnel_MissingFunnelID(t *testing.T) {
	repo := newMockGroupFunnelRepo()
	uc := NewManageGroupFunnelsUseCase(repo)

	err := uc.SetGroupFunnel(context.Background(), GroupFunnelInput{
		GroupID:  "group-1",
		FunnelID: "",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelIDRequired)
}

func TestManageGroupFunnels_ListByGroup_Success(t *testing.T) {
	gf1 := newTestGroupFunnel("gf-1", "group-1", "funnel-1", []string{"col-1"})
	gf2 := newTestGroupFunnel("gf-2", "group-1", "funnel-2", []string{})
	gf3 := newTestGroupFunnel("gf-3", "group-2", "funnel-1", []string{})
	repo := newMockGroupFunnelRepo(gf1, gf2, gf3)
	uc := NewManageGroupFunnelsUseCase(repo)

	out, err := uc.ListByGroup(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
	ids := []string{out[0].ID, out[1].ID}
	assert.Contains(t, ids, "gf-1")
	assert.Contains(t, ids, "gf-2")
}

func TestManageGroupFunnels_ListByGroup_Empty(t *testing.T) {
	repo := newMockGroupFunnelRepo()
	uc := NewManageGroupFunnelsUseCase(repo)

	out, err := uc.ListByGroup(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, out, 0)
}

func TestManageGroupFunnels_RemoveGroupFunnel_Success(t *testing.T) {
	gf := newTestGroupFunnel("gf-1", "group-1", "funnel-1", []string{"col-1"})
	repo := newMockGroupFunnelRepo(gf)
	uc := NewManageGroupFunnelsUseCase(repo)

	err := uc.RemoveGroupFunnel(context.Background(), "group-1", "funnel-1")

	require.NoError(t, err)
	assert.Len(t, repo.funnels, 0)
}

func TestManageGroupFunnels_RemoveGroupFunnel_NotFound(t *testing.T) {
	repo := newMockGroupFunnelRepo()
	uc := NewManageGroupFunnelsUseCase(repo)

	err := uc.RemoveGroupFunnel(context.Background(), "group-1", "nonexistent")

	assert.ErrorIs(t, err, domain.ErrGroupFunnelNotFound)
}
