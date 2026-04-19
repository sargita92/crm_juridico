package infrastructure

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

// ServiceCheck é uma função de saúde para um serviço externo (ex: MySQL, WhatsApp).
// O wiring (Task 11) injeta as checagens concretas; o provider apenas as executa
// para evitar acoplamento com módulos de infra específicos.
type ServiceCheck func(ctx context.Context) (name string, up bool)

// PrometheusStatsProvider implementa application.InfraProvider lendo métricas do
// registrador Prometheus em uso pela API (ver internal/shared/middleware/prometheus.go).
type PrometheusStatsProvider struct {
	gatherer prometheus.Gatherer
	checks   []ServiceCheck
}

// Compile-time check.
var _ application.InfraProvider = (*PrometheusStatsProvider)(nil)

// NewPrometheusStatsProvider — gatherer nil resolve para prometheus.DefaultGatherer.
func NewPrometheusStatsProvider(gatherer prometheus.Gatherer, checks []ServiceCheck) *PrometheusStatsProvider {
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}
	return &PrometheusStatsProvider{gatherer: gatherer, checks: checks}
}

// Snapshot agrega métricas atuais de latência e taxa de 5xx, e executa as checagens
// de serviço. Métricas usadas:
//   - http_request_duration_seconds (histogram) → APILatencyMs = (sum/count) * 1000
//   - http_requests_total{status} (counter) → Error5xxRate = sum(5xx) / sum(total)
//
// Sem amostras → ambos ficam em 0 (zero-value float64).
func (p *PrometheusStatsProvider) Snapshot(ctx context.Context) (*domain.Infrastructure, error) {
	mfs, err := p.gatherer.Gather()
	if err != nil {
		return nil, err
	}
	out := &domain.Infrastructure{}
	var totalReq, total5xx float64
	var sumLat, countLat float64
	for _, mf := range mfs {
		switch mf.GetName() {
		case "http_requests_total":
			for _, m := range mf.GetMetric() {
				v := m.GetCounter().GetValue()
				totalReq += v
				for _, lp := range m.GetLabel() {
					if lp.GetName() == "status" && len(lp.GetValue()) > 0 && lp.GetValue()[0] == '5' {
						total5xx += v
						break
					}
				}
			}
		case "http_request_duration_seconds":
			for _, m := range mf.GetMetric() {
				h := m.GetHistogram()
				sumLat += h.GetSampleSum()
				countLat += float64(h.GetSampleCount())
			}
		}
	}
	if countLat > 0 {
		out.APILatencyMs = (sumLat / countLat) * 1000
	}
	if totalReq > 0 {
		out.Error5xxRate = total5xx / totalReq
	}
	for _, ck := range p.checks {
		name, up := ck(ctx)
		out.ServicesStatus = append(out.ServicesStatus, domain.ServiceStatus{Name: name, Up: up})
	}
	return out, nil
}
