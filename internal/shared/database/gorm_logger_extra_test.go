package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gormlogger "gorm.io/gorm/logger"
)

// Os níveis de passthrough (Info/Warn/Error) roteiam para o zap no mesmo nível,
// com os campos de contexto anexados. LogMode devolve o próprio logger (o nível
// efetivo é do zap).
func TestZapGormLogger_PassthroughLevels(t *testing.T) {
	log, logs := newObservedLogger()
	ctxFields := func(context.Context) []zap.Field { return []zap.Field{zap.String("request_id", "r1")} }
	l := newZapGormLogger(log, 200*time.Millisecond, ctxFields)

	assert.NotNil(t, l.LogMode(gormlogger.Info))

	ctx := context.Background()
	l.Info(ctx, "info %d", 1)
	l.Warn(ctx, "warn %s", "x")
	l.Error(ctx, "boom")

	infos := logs.FilterLevelExact(zapcore.InfoLevel).All()
	assert.Len(t, infos, 1)
	assert.Equal(t, "info 1", infos[0].Message)
	assert.Equal(t, "r1", infos[0].ContextMap()["request_id"])
	assert.Len(t, logs.FilterLevelExact(zapcore.WarnLevel).All(), 1)
	assert.Len(t, logs.FilterLevelExact(zapcore.ErrorLevel).All(), 1)
}

func TestTruncateSQL(t *testing.T) {
	short := "SELECT 1"
	assert.Equal(t, short, truncateSQL(short))

	long := strings.Repeat("a", maxLoggedSQLLen+100)
	out := truncateSQL(long)
	assert.True(t, strings.HasSuffix(out, "…(truncated)"))
	assert.Equal(t, maxLoggedSQLLen, len(strings.TrimSuffix(out, "…(truncated)")))
}
