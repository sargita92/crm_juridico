package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
)

func TestDefaultPublisher_Success_ReturnsNil(t *testing.T) {
	repo := newMockAuditRepo()
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())
	pub := NewPublisher(uc, zap.NewNop())

	err := pub.Publish(context.Background(), validRegisterInput())

	require.NoError(t, err)
	assert.Equal(t, 1, repo.createCalled)
}

func TestDefaultPublisher_UseCaseError_SwallowsAndLogsWarn(t *testing.T) {
	repo := newMockAuditRepo()
	repo.createErr = errors.New("db down")

	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)

	// UC tem seu proprio logger nop — queremos isolar o WARN do publisher.
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())
	pub := NewPublisher(uc, logger)

	err := pub.Publish(context.Background(), validRegisterInput())

	// Decisao do design: falha de auditoria NAO quebra a operacao.
	require.NoError(t, err)

	require.GreaterOrEqual(t, recorded.Len(), 1)
	entry := recorded.All()[0]
	assert.Equal(t, zap.WarnLevel, entry.Level)
	assert.Equal(t, "audit publisher swallowed error", entry.Message)
	// Campos de contexto esperados.
	fields := entry.ContextMap()
	assert.Equal(t, string(domain.ActionLoginSuccess), fields["action"])
	assert.Equal(t, "admin@crm.com", fields["actor_email"])
}

func TestNoopPublisher_AlwaysNil_DoesNotTouchRepo(t *testing.T) {
	repo := newMockAuditRepo()
	pub := NoopPublisher{}

	err := pub.Publish(context.Background(), validRegisterInput())

	require.NoError(t, err)
	assert.Equal(t, 0, repo.createCalled, "Noop nunca persiste")
}
