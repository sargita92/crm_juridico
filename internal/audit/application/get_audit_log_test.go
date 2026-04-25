package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

func TestGetAuditLog_HappyPath_ReturnsLog(t *testing.T) {
	repo := newMockAuditRepo()
	want := sampleAuditLog(t)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.AuditLog, error) {
		assert.Equal(t, want.ID, id)
		return want, nil
	}
	uc := NewGetAuditLogUseCase(repo, zap.NewNop())

	got, err := uc.Execute(context.Background(), want.ID)

	require.NoError(t, err)
	assert.Same(t, want, got)
}

func TestGetAuditLog_EmptyID_ReturnsNotFoundWithoutCallingRepo(t *testing.T) {
	repo := newMockAuditRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.AuditLog, error) {
		t.Fatalf("repo.FindByID nao deveria ser chamado para id vazio")
		return nil, nil
	}
	uc := NewGetAuditLogUseCase(repo, zap.NewNop())

	got, err := uc.Execute(context.Background(), "   ")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAuditLogNotFound)
	assert.Nil(t, got)
}

func TestGetAuditLog_RepoReturnsNotFound_PropagatesError(t *testing.T) {
	repo := newMockAuditRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.AuditLog, error) {
		return nil, domain.ErrAuditLogNotFound
	}
	uc := NewGetAuditLogUseCase(repo, zap.NewNop())

	got, err := uc.Execute(context.Background(), "id-que-nao-existe")

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAuditLogNotFound)
	assert.Nil(t, got)
}

func TestGetAuditLog_RepoReturnsOtherError_Propagates(t *testing.T) {
	repo := newMockAuditRepo()
	repoErr := errors.New("connection refused")
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.AuditLog, error) {
		return nil, repoErr
	}
	uc := NewGetAuditLogUseCase(repo, zap.NewNop())

	got, err := uc.Execute(context.Background(), "any-id")

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, got)
}

func TestGetAuditLog_ContextCanceled_PropagatesError(t *testing.T) {
	repo := newMockAuditRepo()
	repo.findByIDFn = func(ctx context.Context, _ string) (*domain.AuditLog, error) {
		return nil, ctx.Err()
	}
	uc := NewGetAuditLogUseCase(repo, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.Execute(ctx, "any-id")

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
