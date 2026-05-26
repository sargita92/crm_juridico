package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/sasrgita/crm-juridico/internal/shared/database"
)

// TestEnableQueryTracing_EmitsChildSpanPerQuery garante que cada query do Gorm
// emite um span OTel filho do span do request — base para localizar, no trace,
// qual query consome o tempo (F26 H5).
func TestEnableQueryTracing_EmitsChildSpanPerQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	db := sharedContainer.DB(t) // conexão nova e isolada por teste

	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	require.NoError(t, database.EnableQueryTracing(db, tp))

	ctx, parent := tp.Tracer("test").Start(context.Background(), "parent-request")
	var n int
	require.NoError(t, db.WithContext(ctx).Raw("SELECT 1").Scan(&n).Error)
	parent.End()

	var dbSpans []sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if strings.HasPrefix(s.Name(), "gorm.") {
			dbSpans = append(dbSpans, s)
		}
	}
	require.NotEmpty(t, dbSpans, "esperava ao menos um span gorm.* para a query")

	span := dbSpans[0]
	require.Equal(t, parent.SpanContext().SpanID(), span.Parent().SpanID(),
		"o span da query deve ser filho do span do request")

	var stmt string
	for _, attr := range span.Attributes() {
		if attr.Key == "db.statement" {
			stmt = attr.Value.AsString()
		}
	}
	require.Contains(t, stmt, "SELECT 1", "o span deve registrar db.statement")
}
