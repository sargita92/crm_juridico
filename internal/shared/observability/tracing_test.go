package observability_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

func newInMemoryTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	return sr
}

func TestStartSpan_CreatesSpanWithName(t *testing.T) {
	sr := newInMemoryTracer(t)

	_, span := observability.StartSpan(context.Background(), "automation.usecase.trigger")
	span.End()

	snapshots := sr.Ended()
	assert.Len(t, snapshots, 1)
	assert.Equal(t, "automation.usecase.trigger", snapshots[0].Name())
}

func TestStartSpan_AttachesAttributes(t *testing.T) {
	sr := newInMemoryTracer(t)

	_, span := observability.StartSpan(
		context.Background(),
		"permission.usecase.check",
		attribute.String("tenant.id", "t-42"),
		attribute.String("user.id", "u-9"),
	)
	span.End()

	snapshots := sr.Ended()
	attrs := snapshots[0].Attributes()
	var tenantID, userID string
	for _, a := range attrs {
		if a.Key == "tenant.id" {
			tenantID = a.Value.AsString()
		}
		if a.Key == "user.id" {
			userID = a.Value.AsString()
		}
	}
	assert.Equal(t, "t-42", tenantID)
	assert.Equal(t, "u-9", userID)
}
