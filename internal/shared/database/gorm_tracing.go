package database

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// EnableQueryTracing registra callbacks no Gorm que emitem um span OTel por
// query, filho do span do request presente em db.Statement.Context. Isso torna
// o tempo gasto em banco visível no trace end-to-end (regra de observabilidade
// do projeto; F26). Quando tp é nil, usa o tracer provider global.
//
// O atributo db.statement guarda o SQL com placeholders (?), não os valores —
// evitando vazar dados sensíveis nos traces.
func EnableQueryTracing(db *gorm.DB, tp trace.TracerProvider) error {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}
	g := &gormTracer{tracer: tp.Tracer("crm_juridico/gorm")}
	return g.register(db)
}

type gormTracer struct {
	tracer trace.Tracer
}

func (g *gormTracer) before(spanName string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}
		ctx, _ := g.tracer.Start(db.Statement.Context, spanName)
		db.Statement.Context = ctx
	}
}

func (g *gormTracer) after() func(*gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement == nil || db.Statement.Context == nil {
			return
		}
		span := trace.SpanFromContext(db.Statement.Context)
		if !span.IsRecording() {
			return
		}
		defer span.End()

		span.SetAttributes(
			attribute.String("db.system", "mysql"),
			attribute.String("db.statement", db.Statement.SQL.String()),
			attribute.Int64("db.rows_affected", db.Statement.RowsAffected),
		)
		if db.Statement.Table != "" {
			span.SetAttributes(attribute.String("db.sql.table", db.Statement.Table))
		}
		// ErrRecordNotFound é fluxo de domínio normal, não erro de infraestrutura.
		if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
			span.RecordError(db.Error)
			span.SetStatus(codes.Error, db.Error.Error())
		}
	}
}

// register liga os callbacks before/after nas seis operações do Gorm. O anchor
// "gorm:raw" não existe por padrão (Raw não tem callback default), mas o Gorm
// apenas anexa nesse caso — sem erro.
func (g *gormTracer) register(db *gorm.DB) error {
	type hook struct {
		register interface {
			Register(name string, fn func(*gorm.DB)) error
		}
		fn   func(*gorm.DB)
		name string
	}
	cb := db.Callback()
	after := g.after()
	hooks := []hook{
		{cb.Create().Before("gorm:create"), g.before("gorm.Create"), "before:create"},
		{cb.Create().After("gorm:create"), after, "after:create"},
		{cb.Query().Before("gorm:query"), g.before("gorm.Query"), "before:query"},
		{cb.Query().After("gorm:query"), after, "after:query"},
		{cb.Update().Before("gorm:update"), g.before("gorm.Update"), "before:update"},
		{cb.Update().After("gorm:update"), after, "after:update"},
		{cb.Delete().Before("gorm:delete"), g.before("gorm.Delete"), "before:delete"},
		{cb.Delete().After("gorm:delete"), after, "after:delete"},
		{cb.Row().Before("gorm:row"), g.before("gorm.Row"), "before:row"},
		{cb.Row().After("gorm:row"), after, "after:row"},
		{cb.Raw().Before("gorm:raw"), g.before("gorm.Raw"), "before:raw"},
		{cb.Raw().After("gorm:raw"), after, "after:raw"},
	}

	var firstErr error
	for _, h := range hooks {
		if err := h.register.Register("otel:"+h.name, h.fn); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("register gorm tracing callback %s: %w", h.name, err)
		}
	}
	return firstErr
}
