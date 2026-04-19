package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func TestGetFileUseCase_EmptyTenant(t *testing.T) {
	uc := NewGetFileUseCase(newMockFileRepo())
	_, err := uc.Execute(context.Background(), "", "id")
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestGetFileUseCase_NotFound(t *testing.T) {
	uc := NewGetFileUseCase(newMockFileRepo())
	_, err := uc.Execute(context.Background(), "t1", "missing")
	assert.ErrorIs(t, err, domain.ErrFileNotFound)
}

func TestGetFileUseCase_CrossTenantNotFound(t *testing.T) {
	repo := newMockFileRepo()
	f, _ := domain.NewFile(
		"id-1", "tA", "c1", "k1", "x.pdf", "application/pdf", "k",
		1, domain.MediaTypeDocument, domain.DirectionInbound, nil, nil,
	)
	require.NoError(t, repo.Create(context.Background(), f))

	uc := NewGetFileUseCase(repo)
	_, err := uc.Execute(context.Background(), "tB", "id-1")
	assert.ErrorIs(t, err, domain.ErrFileNotFound)
}

func TestGetFileUseCase_Found(t *testing.T) {
	repo := newMockFileRepo()
	f, _ := domain.NewFile(
		"id-1", "t1", "c1", "k1", "x.pdf", "application/pdf", "k",
		1, domain.MediaTypeDocument, domain.DirectionInbound, nil, nil,
	)
	require.NoError(t, repo.Create(context.Background(), f))

	uc := NewGetFileUseCase(repo)
	got, err := uc.Execute(context.Background(), "t1", "id-1")
	require.NoError(t, err)
	assert.Equal(t, "x.pdf", got.Name)
}
