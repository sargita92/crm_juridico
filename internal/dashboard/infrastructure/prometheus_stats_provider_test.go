package infrastructure_test

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/dashboard/infrastructure"
)

func newRegistryWithSamples(t *testing.T) *prometheus.Registry {
	t.Helper()
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "test"},
		[]string{"method", "path", "status"},
	)
	hist := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "test", Buckets: prometheus.DefBuckets},
		[]string{"method", "path", "status"},
	)
	require.NoError(t, reg.Register(counter))
	require.NoError(t, reg.Register(hist))

	// 8 reqs 200 + 2 reqs 500 → total=10, 5xx=2 → rate=0.2
	for i := 0; i < 8; i++ {
		counter.WithLabelValues("GET", "/x", "200").Inc()
		hist.WithLabelValues("GET", "/x", "200").Observe(0.05) // 50ms
	}
	for i := 0; i < 2; i++ {
		counter.WithLabelValues("GET", "/x", "500").Inc()
		hist.WithLabelValues("GET", "/x", "500").Observe(0.10) // 100ms
	}
	// Esperado: APILatencyMs = (8*0.05 + 2*0.10) / 10 * 1000 = 0.6/10*1000 = 60.0
	return reg
}

func TestPrometheusStatsProvider_Snapshot_LatencyAndErrorRate(t *testing.T) {
	reg := newRegistryWithSamples(t)
	p := infrastructure.NewPrometheusStatsProvider(reg, nil)

	got, err := p.Snapshot(context.Background())
	require.NoError(t, err)
	assert.InDelta(t, 60.0, got.APILatencyMs, 0.01)
	assert.InDelta(t, 0.2, got.Error5xxRate, 0.001)
	assert.Empty(t, got.ServicesStatus) // sem checks injetados
}

func TestPrometheusStatsProvider_Snapshot_EmptyRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := infrastructure.NewPrometheusStatsProvider(reg, nil)
	got, err := p.Snapshot(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0.0, got.APILatencyMs)
	assert.Equal(t, 0.0, got.Error5xxRate)
	assert.Empty(t, got.ServicesStatus)
}

func TestPrometheusStatsProvider_Snapshot_RunsServiceChecks(t *testing.T) {
	reg := prometheus.NewRegistry()
	checks := []infrastructure.ServiceCheck{
		func(ctx context.Context) (string, bool) { return "mysql", true },
		func(ctx context.Context) (string, bool) { return "whatsapp", false },
	}
	p := infrastructure.NewPrometheusStatsProvider(reg, checks)

	got, err := p.Snapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, got.ServicesStatus, 2)
	assert.Equal(t, "mysql", got.ServicesStatus[0].Name)
	assert.True(t, got.ServicesStatus[0].Up)
	assert.Equal(t, "whatsapp", got.ServicesStatus[1].Name)
	assert.False(t, got.ServicesStatus[1].Up)
}

func TestPrometheusStatsProvider_Snapshot_GathererError(t *testing.T) {
	p := infrastructure.NewPrometheusStatsProvider(failingGatherer{}, nil)
	_, err := p.Snapshot(context.Background())
	require.Error(t, err)
}

type failingGatherer struct{}

func (failingGatherer) Gather() ([]*dto.MetricFamily, error) { return nil, errors.New("boom") }
