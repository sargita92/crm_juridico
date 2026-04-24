package observability_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

func TestInitTracer_FallbacksToStdoutWhenEndpointUnset(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	tp, err := observability.InitTracer("crm-juridico-test")
	assert.NoError(t, err)
	assert.NotNil(t, tp)
	_ = tp.Shutdown(t.Context())
}

func TestInitTracer_UsesOTLPWhenEndpointSet(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	tp, err := observability.InitTracer("crm-juridico-test")
	assert.NoError(t, err)
	assert.NotNil(t, tp)
	_ = tp.Shutdown(t.Context())
}
