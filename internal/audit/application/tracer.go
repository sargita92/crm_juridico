package application

import "go.opentelemetry.io/otel"

// tracer e o tracer OpenTelemetry compartilhado pelos casos de uso do
// dominio audit. Centralizado em um arquivo para manter o nome do tracer
// consistente nos spans ("audit") que aparecerao no Jaeger/Tempo.
//
// Spans esperados (secao 9.3 do design F12):
//   - audit.usecase.register
//   - audit.usecase.list
//   - audit.usecase.get
var tracer = otel.Tracer("audit")
