package http

import "go.opentelemetry.io/otel"

// httpTracer cria spans no boundary HTTP do dashboard, posicionando-se como
// pai dos spans criados pelos use cases (tracer "dashboard").
var httpTracer = otel.Tracer("dashboard.http")
