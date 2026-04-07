package application

import (
	"context"
	"testing"
	"time"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestViewProfile(id, groupID, funnelID string, cols []string) domain.ViewProfile {
	return domain.ViewProfile{
		ID:             id,
		GroupID:        groupID,
		FunnelID:       funnelID,
		VisibleColumns: cols,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func TestManageViewProfiles_SetViewProfile_Create(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	err := uc.SetViewProfile(context.Background(), ViewProfileInput{
		GroupID:        "group-1",
		FunnelID:       "funnel-1",
		VisibleColumns: []string{"col-1", "col-2"},
	})

	require.NoError(t, err)
	assert.Len(t, repo.profiles, 1)
	assert.Equal(t, []string{"col-1", "col-2"}, repo.profiles[0].VisibleColumns)
}

func TestManageViewProfiles_SetViewProfile_Update(t *testing.T) {
	existing := newTestViewProfile("vp-1", "group-1", "funnel-1", []string{"col-1"})
	repo := newMockViewProfileRepo(existing)
	uc := NewManageViewProfilesUseCase(repo)

	err := uc.SetViewProfile(context.Background(), ViewProfileInput{
		GroupID:        "group-1",
		FunnelID:       "funnel-1",
		VisibleColumns: []string{"col-1", "col-2", "col-3"},
	})

	require.NoError(t, err)
	assert.Len(t, repo.profiles, 1, "should upsert, not append")
	assert.Equal(t, []string{"col-1", "col-2", "col-3"}, repo.profiles[0].VisibleColumns)
}

func TestManageViewProfiles_SetViewProfile_MissingGroupID(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	err := uc.SetViewProfile(context.Background(), ViewProfileInput{
		GroupID:  "",
		FunnelID: "funnel-1",
	})

	assert.ErrorIs(t, err, domain.ErrGroupIDRequired)
}

func TestManageViewProfiles_SetViewProfile_MissingFunnelID(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	err := uc.SetViewProfile(context.Background(), ViewProfileInput{
		GroupID:  "group-1",
		FunnelID: "",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelIDRequired)
}

func TestManageViewProfiles_ListByGroup_Success(t *testing.T) {
	vp1 := newTestViewProfile("vp-1", "group-1", "funnel-1", []string{"col-1"})
	vp2 := newTestViewProfile("vp-2", "group-1", "funnel-2", []string{"col-2"})
	vp3 := newTestViewProfile("vp-3", "group-2", "funnel-1", []string{})
	repo := newMockViewProfileRepo(vp1, vp2, vp3)
	uc := NewManageViewProfilesUseCase(repo)

	out, err := uc.ListByGroup(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
	ids := []string{out[0].ID, out[1].ID}
	assert.Contains(t, ids, "vp-1")
	assert.Contains(t, ids, "vp-2")
}

func TestManageViewProfiles_ListByGroup_Empty(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	out, err := uc.ListByGroup(context.Background(), "group-1")

	require.NoError(t, err)
	assert.Len(t, out, 0)
}

func TestManageViewProfiles_ResolveVisibleColumns_NoGroups(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	cols, err := uc.ResolveVisibleColumns(context.Background(), []string{}, "funnel-1")

	require.NoError(t, err)
	assert.Nil(t, cols, "no groups means no restrictions — show all")
}

func TestManageViewProfiles_ResolveVisibleColumns_NoProfiles(t *testing.T) {
	repo := newMockViewProfileRepo()
	uc := NewManageViewProfilesUseCase(repo)

	cols, err := uc.ResolveVisibleColumns(context.Background(), []string{"group-1"}, "funnel-1")

	require.NoError(t, err)
	assert.Nil(t, cols, "no profiles configured means no restrictions — show all")
}

func TestManageViewProfiles_ResolveVisibleColumns_Union(t *testing.T) {
	// group-1 sees col-1 and col-2; group-2 sees col-2 and col-3 — union = col-1, col-2, col-3
	vp1 := newTestViewProfile("vp-1", "group-1", "funnel-1", []string{"col-1", "col-2"})
	vp2 := newTestViewProfile("vp-2", "group-2", "funnel-1", []string{"col-2", "col-3"})
	repo := newMockViewProfileRepo(vp1, vp2)
	uc := NewManageViewProfilesUseCase(repo)

	cols, err := uc.ResolveVisibleColumns(context.Background(), []string{"group-1", "group-2"}, "funnel-1")

	require.NoError(t, err)
	assert.Len(t, cols, 3)
	assert.Contains(t, cols, "col-1")
	assert.Contains(t, cols, "col-2")
	assert.Contains(t, cols, "col-3")
}

func TestManageViewProfiles_ResolveVisibleColumns_FunnelFilter(t *testing.T) {
	// Profile exists for funnel-2, not funnel-1 — should return nil (no restrictions)
	vp := newTestViewProfile("vp-1", "group-1", "funnel-2", []string{"col-1"})
	repo := newMockViewProfileRepo(vp)
	uc := NewManageViewProfilesUseCase(repo)

	cols, err := uc.ResolveVisibleColumns(context.Background(), []string{"group-1"}, "funnel-1")

	require.NoError(t, err)
	assert.Nil(t, cols, "no profiles for requested funnel means no restrictions")
}

func TestManageViewProfiles_ResolveVisibleColumns_EmptyColumns(t *testing.T) {
	// Group has a profile with empty visible columns
	vp := newTestViewProfile("vp-1", "group-1", "funnel-1", []string{})
	repo := newMockViewProfileRepo(vp)
	uc := NewManageViewProfilesUseCase(repo)

	cols, err := uc.ResolveVisibleColumns(context.Background(), []string{"group-1"}, "funnel-1")

	require.NoError(t, err)
	// Profile exists but has no visible columns — union returns empty (non-nil) slice
	assert.NotNil(t, cols)
	assert.Len(t, cols, 0)
}
