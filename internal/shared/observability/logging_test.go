package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

func TestLoggerFromContext_ReturnsBaseWhenNoSpan(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	base := zap.New(core)

	out := observability.LoggerFromContext(context.Background(), base)
	out.Info("hello")

	entries := recorded.All()
	assert.Len(t, entries, 1)
	for _, f := range entries[0].Context {
		assert.NotEqual(t, "trace_id", f.Key, "nao deve injetar trace_id sem span")
	}
}

func TestLoggerFromContext_InjectsTraceAndSpanID(t *testing.T) {
	_ = newInMemoryTracer(t)
	core, recorded := observer.New(zap.InfoLevel)
	base := zap.New(core)

	ctx, span := observability.StartSpan(context.Background(), "test.usecase.run")
	out := observability.LoggerFromContext(ctx, base)
	out.Info("hello")
	span.End()

	entries := recorded.All()
	assert.Len(t, entries, 1)
	var traceID, spanID string
	for _, f := range entries[0].Context {
		if f.Key == "trace_id" {
			traceID = f.String
		}
		if f.Key == "span_id" {
			spanID = f.String
		}
	}
	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)
}
