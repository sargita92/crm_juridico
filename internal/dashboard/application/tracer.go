package application

import "go.opentelemetry.io/otel"

var tracer = otel.Tracer("dashboard")
