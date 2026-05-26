package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// ContextFieldExtractor extrai campos de log (ex.: request_id, tenant_id) do
// contexto do request. É injetado em New para que o pacote database permaneça
// desacoplado do pacote middleware (evita ciclo de import) e seja testável.
type ContextFieldExtractor func(context.Context) []zap.Field

// maxLoggedSQLLen limita o tamanho do SQL gravado no log de slow query para
// evitar entradas gigantes (ex.: IN com centenas de ids).
const maxLoggedSQLLen = 2048

// zapGormLogger implementa gorm/logger.Interface roteando para zap. Só registra
// queries com erro ou acima do limiar de lentidão — queries rápidas não geram
// log, mantendo o sinal limpo para diagnosticar gargalos (F26).
type zapGormLogger struct {
	log           *zap.Logger
	slowThreshold time.Duration
	ctxFields     ContextFieldExtractor
}

func newZapGormLogger(log *zap.Logger, slowThreshold time.Duration, ctxFields ContextFieldExtractor) *zapGormLogger {
	if ctxFields == nil {
		ctxFields = func(context.Context) []zap.Field { return nil }
	}
	return &zapGormLogger{log: log, slowThreshold: slowThreshold, ctxFields: ctxFields}
}

// LogMode satisfaz a interface; o nível efetivo é controlado pelo zap subjacente.
func (l *zapGormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (l *zapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.log.With(l.ctxFields(ctx)...).Info(fmt.Sprintf(msg, data...))
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.log.With(l.ctxFields(ctx)...).Warn(fmt.Sprintf(msg, data...))
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.log.With(l.ctxFields(ctx)...).Error(fmt.Sprintf(msg, data...))
}

// Trace é chamado pelo Gorm ao final de cada query. Registra erros reais e
// queries lentas; ignora ErrRecordNotFound (fluxo de domínio) e queries rápidas.
func (l *zapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)

	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		l.log.With(l.ctxFields(ctx)...).Error("db query error",
			zap.String("sql", truncateSQL(sql)),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	case l.slowThreshold > 0 && elapsed > l.slowThreshold:
		sql, rows := fc()
		l.log.With(l.ctxFields(ctx)...).Warn("slow query",
			zap.String("sql", truncateSQL(sql)),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.Duration("threshold", l.slowThreshold),
		)
	}
}

func truncateSQL(sql string) string {
	if len(sql) > maxLoggedSQLLen {
		return sql[:maxLoggedSQLLen] + "…(truncated)"
	}
	return sql
}
