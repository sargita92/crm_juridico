package application

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
)

func validRegisterInput() RegisterAuditLogInput {
	return RegisterAuditLogInput{
		ActorEmail: "admin@crm.com",
		Action:     domain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
		UserAgent:  "Mozilla/5.0",
	}
}

// counterValue le o valor atual de um CounterVec para um conjunto de labels.
//
// Usa o protocolo de coleta do Prometheus em vez de chamar metodos privados.
// Retorna 0 quando o counter ainda nao foi incrementado para esses labels.
func counterValue(t *testing.T, cv *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()
	c, err := cv.GetMetricWithLabelValues(labels...)
	require.NoError(t, err)
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}

func TestRegisterAuditLog_HappyPath_PersistsAndIncrementsSuccess(t *testing.T) {
	repo := newMockAuditRepo()
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())

	before := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		string(domain.ActionLoginSuccess), "success")

	err := uc.Execute(context.Background(), validRegisterInput())

	require.NoError(t, err)
	require.Equal(t, 1, repo.createCalled)
	got := repo.lastCreated()
	require.NotNil(t, got)
	assert.Equal(t, "admin@crm.com", got.ActorEmail)
	assert.Equal(t, domain.ActionLoginSuccess, got.Action)
	assert.Equal(t, "session", got.Entity)
	assert.NotEmpty(t, got.ID)

	after := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		string(domain.ActionLoginSuccess), "success")
	assert.InDelta(t, before+1, after, 0.0001)
}

func TestRegisterAuditLog_InvalidAction_ReturnsErrorAndIncrementsErrorCounter(t *testing.T) {
	repo := newMockAuditRepo()
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())

	in := validRegisterInput()
	in.Action = domain.Action("not_a_real_action")

	before := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		"not_a_real_action", "error")

	err := uc.Execute(context.Background(), in)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAction)
	assert.Equal(t, 0, repo.createCalled, "nao deve persistir input invalido")

	after := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		"not_a_real_action", "error")
	assert.InDelta(t, before+1, after, 0.0001)
}

func TestRegisterAuditLog_EmptyActorEmail_ReturnsDomainError(t *testing.T) {
	repo := newMockAuditRepo()
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())

	in := validRegisterInput()
	in.ActorEmail = ""

	err := uc.Execute(context.Background(), in)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrActorEmailRequired)
	assert.Equal(t, 0, repo.createCalled)
}

func TestRegisterAuditLog_RepoError_PropagatesAndLogsWarn(t *testing.T) {
	repo := newMockAuditRepo()
	repo.createErr = errors.New("connection refused")

	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	uc := NewRegisterAuditLogUseCase(repo, logger)

	before := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		string(domain.ActionLoginSuccess), "error")

	err := uc.Execute(context.Background(), validRegisterInput())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")

	after := counterValue(t, auditinfra.AuditLogsRegisteredTotal,
		string(domain.ActionLoginSuccess), "error")
	assert.InDelta(t, before+1, after, 0.0001)

	require.Equal(t, 1, recorded.Len(), "deve ter logado WARN ao falhar persist")
	entry := recorded.All()[0]
	assert.Equal(t, "audit log persist failed", entry.Message)
	assert.Equal(t, zap.WarnLevel, entry.Level)
}

func TestRegisterAuditLog_MetadataWithForbiddenKey_SanitizesBeforePersist(t *testing.T) {
	repo := newMockAuditRepo()
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())

	in := validRegisterInput()
	in.Metadata = domain.Metadata{
		"reason":        "manual_block",
		"password":      "leaked-secret",
		"Password_Hash": "argon2id$...",
		"AUTHORIZATION": "Bearer xyz",
	}

	err := uc.Execute(context.Background(), in)
	require.NoError(t, err)

	got := repo.lastCreated()
	require.NotNil(t, got)
	assert.Contains(t, got.Metadata, "reason")
	assert.NotContains(t, got.Metadata, "password")
	assert.NotContains(t, got.Metadata, "Password_Hash")
	assert.NotContains(t, got.Metadata, "AUTHORIZATION")
}

func TestRegisterAuditLog_ContextCanceled_PropagatesError(t *testing.T) {
	repo := newMockAuditRepo()
	repo.createErr = context.Canceled
	uc := NewRegisterAuditLogUseCase(repo, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := uc.Execute(ctx, validRegisterInput())

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
