package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "crm_juridico"

// StartSpan cria um novo span no tracer global com o nome fornecido
// e os atributos opcionais. O padrão de nome é "<module>.usecase.<action>"
// para use cases (ex.: "automation.usecase.trigger") e
// "<module>.repo.<action>" para repositórios.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	opts := []trace.SpanStartOption{}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	return tracer.Start(ctx, name, opts...)
}
