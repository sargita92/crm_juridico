package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func seedLeadFile(t *testing.T, repo *mockFileRepo, tenantID, leadID, name string, createdAt time.Time) {
	t.Helper()
	f, err := domain.NewFile(
		"id-"+name, tenantID, "c1", "k1",
		name, "application/octet-stream", "k-"+name, int64(len(name)),
		domain.MediaTypeDocument, domain.DirectionInbound, &leadID, nil,
	)
	require.NoError(t, err)
	f.CreatedAt = createdAt
	f.UpdatedAt = createdAt
	require.NoError(t, repo.Create(context.Background(), f))
}

func TestLeadFilesSummaryUseCase_EmptyTenantRejected(t *testing.T) {
	uc := NewLeadFilesSummaryUseCase(newMockFileRepo())
	_, err := uc.Execute(context.Background(), "", "L")
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestLeadFilesSummaryUseCase_EmptySummary(t *testing.T) {
	uc := NewLeadFilesSummaryUseCase(newMockFileRepo())
	out, err := uc.Execute(context.Background(), "t1", "L")
	require.NoError(t, err)
	assert.Equal(t, int64(0), out.Total)
	assert.Empty(t, out.Recent)
}

func TestLeadFilesSummaryUseCase_TopSix(t *testing.T) {
	repo := newMockFileRepo()
	base := time.Now()
	for i := 0; i < 10; i++ {
		seedLeadFile(t, repo, "t1", "L", "f"+string(rune('0'+i)), base.Add(time.Duration(i)*time.Second))
	}
	uc := NewLeadFilesSummaryUseCase(repo)
	out, err := uc.Execute(context.Background(), "t1", "L")
	require.NoError(t, err)
	assert.Equal(t, int64(10), out.Total)
	assert.Len(t, out.Recent, 6)
	// Newest first
	assert.Equal(t, "f9", out.Recent[0].Name)
}

func TestLeadFilesSummaryUseCase_TenantIsolated(t *testing.T) {
	repo := newMockFileRepo()
	now := time.Now()
	seedLeadFile(t, repo, "tA", "L", "a", now)
	seedLeadFile(t, repo, "tB", "L", "b", now)
	uc := NewLeadFilesSummaryUseCase(repo)
	out, err := uc.Execute(context.Background(), "tA", "L")
	require.NoError(t, err)
	assert.Equal(t, int64(1), out.Total)
	assert.Equal(t, "a", out.Recent[0].Name)
}
