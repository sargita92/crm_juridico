package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// LoggerFromContext retorna um logger derivado de base com os campos
// trace_id e span_id do span ativo em ctx, quando houver. Se não houver
// span ativo, retorna base sem modificação.
func LoggerFromContext(ctx context.Context, base *zap.Logger) *zap.Logger {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return base
	}
	return base.With(
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	)
}
