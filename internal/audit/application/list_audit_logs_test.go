package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
)

// histogramSampleCount le o numero acumulado de amostras do label vazio
// do histograma. Usa o protocolo de coleta do Prometheus em vez de
// inspecionar campos privados.
func histogramSampleCount(t *testing.T) uint64 {
	t.Helper()
	obs, err := auditinfra.AuditLogsListDuration.GetMetricWithLabelValues()
	require.NoError(t, err)
	hist, ok := obs.(interface{ Write(*dto.Metric) error })
	require.True(t, ok, "observador deve implementar Write para inspecao")
	var m dto.Metric
	require.NoError(t, hist.Write(&m))
	return m.GetHistogram().GetSampleCount()
}

func validListFilter() domain.Filter {
	return domain.Filter{
		Page:     1,
		PageSize: 20,
	}
}

func sampleAuditLog(t *testing.T) *domain.AuditLog {
	t.Helper()
	log, err := domain.NewAuditLog(domain.NewAuditLogInput{
		ActorEmail: "admin@crm.com",
		Action:     domain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
	})
	require.NoError(t, err)
	return log
}

func TestListAuditLogs_HappyPath_PassesNormalizedFilterToRepo(t *testing.T) {
	repo := newMockAuditRepo()
	expected := []*domain.AuditLog{sampleAuditLog(t), sampleAuditLog(t)}

	var captured domain.Filter
	repo.listFn = func(_ context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error) {
		captured = f
		return expected, 42, nil
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	logs, total, err := uc.Execute(context.Background(), validListFilter())

	require.NoError(t, err)
	assert.Equal(t, expected, logs)
	assert.Equal(t, int64(42), total)
	assert.Equal(t, 1, captured.Page)
	assert.Equal(t, 20, captured.PageSize)
}

func TestListAuditLogs_InvalidAction_ReturnsErrInvalidActionWithoutCallingRepo(t *testing.T) {
	repo := newMockAuditRepo()
	repo.listFn = func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
		t.Fatalf("repo.List nao deveria ser chamado para action invalida")
		return nil, 0, nil
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	invalid := domain.Action("not_a_real_action")
	filter := validListFilter()
	filter.Action = &invalid

	logs, total, err := uc.Execute(context.Background(), filter)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidAction)
	assert.Nil(t, logs)
	assert.Equal(t, int64(0), total)
}

func TestListAuditLogs_FromAfterTo_ReturnsErrInvalidPeriodWithoutCallingRepo(t *testing.T) {
	repo := newMockAuditRepo()
	repo.listFn = func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
		t.Fatalf("repo.List nao deveria ser chamado para periodo invertido")
		return nil, 0, nil
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	from := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	filter := validListFilter()
	filter.From = &from
	filter.To = &to

	_, _, err := uc.Execute(context.Background(), filter)

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidPeriod)
}

func TestListAuditLogs_PageAndSizeNormalizedBeforeRepo(t *testing.T) {
	repo := newMockAuditRepo()
	var captured domain.Filter
	repo.listFn = func(_ context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error) {
		captured = f
		return nil, 0, nil
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	filter := domain.Filter{
		Page:     -3,
		PageSize: 999,
	}
	_, _, err := uc.Execute(context.Background(), filter)
	require.NoError(t, err)

	assert.Equal(t, 1, captured.Page, "page <= 0 deve virar 1 antes de chegar ao repo")
	assert.Equal(t, domain.MaxPageSize, captured.PageSize,
		"page_size > MaxPageSize deve ser clampado antes de chegar ao repo")
}

func TestListAuditLogs_RepoError_PropagatesToCaller(t *testing.T) {
	repo := newMockAuditRepo()
	repoErr := errors.New("connection refused")
	repo.listFn = func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
		return nil, 0, repoErr
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	logs, total, err := uc.Execute(context.Background(), validListFilter())

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, logs)
	assert.Equal(t, int64(0), total)
}

func TestListAuditLogs_HistogramObservesDuration(t *testing.T) {
	repo := newMockAuditRepo()
	repo.listFn = func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
		return nil, 0, nil
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	beforeCount := histogramSampleCount(t)
	beforeSeries := testutil.CollectAndCount(auditinfra.AuditLogsListDuration)

	_, _, err := uc.Execute(context.Background(), validListFilter())
	require.NoError(t, err)

	afterCount := histogramSampleCount(t)
	afterSeries := testutil.CollectAndCount(auditinfra.AuditLogsListDuration)

	assert.GreaterOrEqual(t, afterSeries, beforeSeries,
		"histograma deve estar coletavel apos Execute")
	assert.Equal(t, beforeCount+1, afterCount,
		"deve haver exatamente uma nova amostra apos Execute")
}

func TestListAuditLogs_ContextCanceled_PropagatesError(t *testing.T) {
	repo := newMockAuditRepo()
	repo.listFn = func(ctx context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
		return nil, 0, ctx.Err()
	}
	uc := NewListAuditLogsUseCase(repo, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := uc.Execute(ctx, validListFilter())

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
