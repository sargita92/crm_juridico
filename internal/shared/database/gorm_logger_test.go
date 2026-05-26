package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
)

func newObservedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core), logs
}

const fakeSQL = "SELECT * FROM leads WHERE tenant_id = 'abc'"

func fakeFC() (string, int64) { return fakeSQL, 3 }

// Acima do limiar, sem erro: loga exatamente um Warn "slow query" com o SQL,
// a duração e os campos de contexto injetados.
func TestZapGormLogger_Trace_SlowQueryLogsWarn(t *testing.T) {
	log, logs := newObservedLogger()
	ctxFields := func(context.Context) []zap.Field { return []zap.Field{zap.String("request_id", "req-123")} }
	l := newZapGormLogger(log, 200*time.Millisecond, ctxFields)

	begin := time.Now().Add(-500 * time.Millisecond) // elapsed ~500ms > 200ms
	l.Trace(context.Background(), begin, fakeFC, nil)

	warns := logs.FilterLevelExact(zapcore.WarnLevel).All()
	require.Len(t, warns, 1, "deveria logar exatamente um Warn de slow query")
	entry := warns[0]
	assert.Equal(t, "slow query", entry.Message)
	fields := entry.ContextMap()
	assert.Equal(t, fakeSQL, fields["sql"])
	assert.Contains(t, fields, "elapsed")
	assert.Equal(t, "req-123", fields["request_id"], "campos de contexto devem ser anexados")
}

// Abaixo do limiar, sem erro: não loga nada (evita ruído).
func TestZapGormLogger_Trace_FastQueryNoLog(t *testing.T) {
	log, logs := newObservedLogger()
	l := newZapGormLogger(log, 200*time.Millisecond, nil)

	begin := time.Now() // elapsed ~0 < 200ms
	l.Trace(context.Background(), begin, fakeFC, nil)

	assert.Empty(t, logs.All(), "query rápida não deve gerar log")
}

// Erro real (não ErrRecordNotFound) loga Error independentemente da duração.
func TestZapGormLogger_Trace_QueryErrorLogsError(t *testing.T) {
	log, logs := newObservedLogger()
	l := newZapGormLogger(log, 200*time.Millisecond, nil)

	l.Trace(context.Background(), time.Now(), fakeFC, errors.New("connection refused"))

	errs := logs.FilterLevelExact(zapcore.ErrorLevel).All()
	require.Len(t, errs, 1)
	assert.Equal(t, fakeSQL, errs[0].ContextMap()["sql"])
}

// ErrRecordNotFound é fluxo normal de domínio e não deve poluir o log de erro.
func TestZapGormLogger_Trace_RecordNotFoundIsNotError(t *testing.T) {
	log, logs := newObservedLogger()
	l := newZapGormLogger(log, 200*time.Millisecond, nil)

	l.Trace(context.Background(), time.Now(), fakeFC, gorm.ErrRecordNotFound)

	assert.Empty(t, logs.FilterLevelExact(zapcore.ErrorLevel).All(),
		"ErrRecordNotFound não deve ser logado como erro")
}

// Limiar 0 desativa o slow-query log mesmo para queries lentas.
func TestZapGormLogger_Trace_ZeroThresholdDisablesSlowLog(t *testing.T) {
	log, logs := newObservedLogger()
	l := newZapGormLogger(log, 0, nil)

	begin := time.Now().Add(-2 * time.Second)
	l.Trace(context.Background(), begin, fakeFC, nil)

	assert.Empty(t, logs.All(), "limiar 0 desativa o slow-query log")
}
