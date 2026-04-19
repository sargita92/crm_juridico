package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func mustSeed(t *testing.T, repo *mockFileRepo, tenantID, name string, mt domain.MediaType, createdAt time.Time) *domain.File {
	t.Helper()
	f, err := domain.NewFile(
		"id-"+name, tenantID, "c1", "k1",
		name, "application/octet-stream", "key-"+name, int64(len(name)),
		mt, domain.DirectionInbound, nil, nil,
	)
	require.NoError(t, err)
	f.CreatedAt = createdAt
	f.UpdatedAt = createdAt
	require.NoError(t, repo.Create(context.Background(), f))
	return f
}

func TestListFilesUseCase_EmptyTenantRejected(t *testing.T) {
	uc := NewListFilesUseCase(newMockFileRepo())
	_, err := uc.Execute(context.Background(), ListFilesInput{})
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestListFilesUseCase_InvalidRange(t *testing.T) {
	uc := NewListFilesUseCase(newMockFileRepo())
	from := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", From: &from, To: &to})
	assert.ErrorIs(t, err, domain.ErrInvalidDateRange)
}

func TestListFilesUseCase_InvalidMediaType(t *testing.T) {
	uc := NewListFilesUseCase(newMockFileRepo())
	mt := domain.MediaType("bogus")
	_, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", MediaType: &mt})
	assert.ErrorIs(t, err, domain.ErrInvalidMediaType)
}

func TestListFilesUseCase_Pagination(t *testing.T) {
	repo := newMockFileRepo()
	base := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	for i, n := range []string{"a", "b", "c", "d", "e"} {
		mustSeed(t, repo, "t1", n, domain.MediaTypeDocument, base.Add(time.Duration(i)*time.Second))
	}
	uc := NewListFilesUseCase(repo)
	res, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), res.Total)
	assert.Len(t, res.Items, 2)
	assert.Equal(t, "e", res.Items[0].Name)
}

func TestListFilesUseCase_PageSizeCapped(t *testing.T) {
	uc := NewListFilesUseCase(newMockFileRepo())
	res, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", PageSize: 9999})
	require.NoError(t, err)
	assert.LessOrEqual(t, res.PageSize, domain.MaxPageSize)
}

func TestListFilesUseCase_FilterByMediaType(t *testing.T) {
	repo := newMockFileRepo()
	base := time.Now()
	mustSeed(t, repo, "t1", "img.jpg", domain.MediaTypeImage, base)
	mustSeed(t, repo, "t1", "doc.pdf", domain.MediaTypeDocument, base)
	mt := domain.MediaTypeImage
	uc := NewListFilesUseCase(repo)
	res, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", MediaType: &mt})
	require.NoError(t, err)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "img.jpg", res.Items[0].Name)
}

func TestListFilesUseCase_FilterByLead(t *testing.T) {
	repo := newMockFileRepo()
	base := time.Now()
	a := mustSeed(t, repo, "t1", "a", domain.MediaTypeDocument, base)
	_ = mustSeed(t, repo, "t1", "b", domain.MediaTypeDocument, base)
	lead := "L"
	a.LeadID = &lead
	repo.byID[a.ID] = a

	uc := NewListFilesUseCase(repo)
	res, err := uc.Execute(context.Background(), ListFilesInput{TenantID: "t1", LeadID: &lead})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Equal(t, "a", res.Items[0].Name)
}
