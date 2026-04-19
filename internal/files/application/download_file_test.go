package application

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func TestDownloadFileUseCase_EmptyTenant(t *testing.T) {
	uc := NewDownloadFileUseCase(newMockFileRepo(), newMockStorage())
	_, _, err := uc.Execute(context.Background(), "", "id")
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestDownloadFileUseCase_CrossTenantNotFound(t *testing.T) {
	repo := newMockFileRepo()
	st := newMockStorage()
	f, _ := domain.NewFile("id-1", "tA", "c1", "k1", "x.pdf", "application/pdf", "key",
		3, domain.MediaTypeDocument, domain.DirectionInbound, nil, nil)
	require.NoError(t, repo.Create(context.Background(), f))

	uc := NewDownloadFileUseCase(repo, st)
	_, _, err := uc.Execute(context.Background(), "tB", "id-1")
	assert.ErrorIs(t, err, domain.ErrFileNotFound)
}

func TestDownloadFileUseCase_ReturnsReader(t *testing.T) {
	repo := newMockFileRepo()
	st := newMockStorage()
	f, _ := domain.NewFile("id-1", "t1", "c1", "k1", "x.pdf", "application/pdf", "key",
		5, domain.MediaTypeDocument, domain.DirectionInbound, nil, nil)
	require.NoError(t, repo.Create(context.Background(), f))
	require.NoError(t, st.Save(context.Background(), "key", []byte("hello")))

	uc := NewDownloadFileUseCase(repo, st)
	got, rc, err := uc.Execute(context.Background(), "t1", "id-1")
	require.NoError(t, err)
	defer rc.Close()
	assert.Equal(t, "x.pdf", got.Name)
	bytes, _ := io.ReadAll(rc)
	assert.Equal(t, "hello", string(bytes))
}

func TestDownloadFileUseCase_MissingContentPropagates(t *testing.T) {
	repo := newMockFileRepo()
	st := newMockStorage()
	f, _ := domain.NewFile("id-1", "t1", "c1", "k1", "x.pdf", "application/pdf", "missing-key",
		1, domain.MediaTypeDocument, domain.DirectionInbound, nil, nil)
	require.NoError(t, repo.Create(context.Background(), f))

	uc := NewDownloadFileUseCase(repo, st)
	_, _, err := uc.Execute(context.Background(), "t1", "id-1")
	assert.ErrorIs(t, err, domain.ErrFileContentUnavailable)
}
