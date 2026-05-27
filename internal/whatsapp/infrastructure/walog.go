package infrastructure

import (
	"fmt"

	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
)

// zapWaLogger adapts whatsmeow's waLog.Logger onto our zap logger so the
// library's internal connection/keepalive/stream diagnostics surface in our
// logs instead of being silently dropped (the client was previously created
// with a nil logger).
type zapWaLogger struct {
	log    *zap.Logger
	module string
}

// newWaLogger returns a waLog.Logger that forwards to log, tagging every entry
// with the whatsmeow module name for filtering.
func newWaLogger(log *zap.Logger, module string) waLog.Logger {
	return &zapWaLogger{log: log, module: module}
}

func (z *zapWaLogger) Errorf(msg string, args ...interface{}) {
	z.log.Error(fmt.Sprintf(msg, args...), zap.String("wa_module", z.module))
}

func (z *zapWaLogger) Warnf(msg string, args ...interface{}) {
	z.log.Warn(fmt.Sprintf(msg, args...), zap.String("wa_module", z.module))
}

func (z *zapWaLogger) Infof(msg string, args ...interface{}) {
	z.log.Info(fmt.Sprintf(msg, args...), zap.String("wa_module", z.module))
}

func (z *zapWaLogger) Debugf(msg string, args ...interface{}) {
	z.log.Debug(fmt.Sprintf(msg, args...), zap.String("wa_module", z.module))
}

func (z *zapWaLogger) Sub(module string) waLog.Logger {
	sub := module
	if z.module != "" {
		sub = z.module + "/" + module
	}
	return &zapWaLogger{log: z.log, module: sub}
}

var _ waLog.Logger = (*zapWaLogger)(nil)
