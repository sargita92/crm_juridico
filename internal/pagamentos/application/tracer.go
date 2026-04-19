package application

import "go.opentelemetry.io/otel"

// tracer is the package-level OpenTelemetry tracer used by all pagamentos
// use cases. Centralizado para manter o nome do tracer consistente nos
// spans ("pagamentos") que aparecerão no Jaeger/Tempo.
var tracer = otel.Tracer("pagamentos")
