package infrastructure

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newObservedProvider(t *testing.T) (*WhatsmeowProvider, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	return NewWhatsmeowProvider(t.TempDir(), zap.New(core)), logs
}

func TestHandleEvent_Connected_SetsGaugeUp(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-connected"

	p.handleEvent(tenant, &events.Connected{})

	assert.Equal(t, 1.0, testutil.ToFloat64(connectionUp.WithLabelValues(tenant)))
	assert.Equal(t, 1, logs.FilterMessage("whatsapp connected").Len())
}

func TestHandleEvent_Disconnected_SetsGaugeDownAndCounts(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-disconnected"
	connectionUp.WithLabelValues(tenant).Set(1)

	p.handleEvent(tenant, &events.Disconnected{})

	assert.Equal(t, 0.0, testutil.ToFloat64(connectionUp.WithLabelValues(tenant)))
	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "disconnected")))
	assert.Equal(t, zapcore.WarnLevel, logs.FilterMessage("whatsapp disconnected").All()[0].Level)
}

func TestHandleEvent_StreamReplaced_Counts(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-streamreplaced"

	p.handleEvent(tenant, &events.StreamReplaced{})

	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "stream_replaced")))
	assert.Equal(t, 1, logs.FilterMessage("whatsapp stream replaced").Len())
}

func TestHandleEvent_StreamError_LogsCode(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-streamerror"

	p.handleEvent(tenant, &events.StreamError{Code: "503"})

	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "stream_error")))
	entry := logs.FilterMessage("whatsapp stream error").All()[0]
	assert.Equal(t, zapcore.ErrorLevel, entry.Level)
	assert.Equal(t, "503", entry.ContextMap()["code"])
}

func TestHandleEvent_ConnectFailure_Counts(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-connectfailure"

	p.handleEvent(tenant, &events.ConnectFailure{Message: "bad"})

	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "connect_failure")))
	assert.Equal(t, 1, logs.FilterMessage("whatsapp connect failure").Len())
}

func TestHandleEvent_TemporaryBan_Counts(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-tempban"

	p.handleEvent(tenant, &events.TemporaryBan{Code: events.TempBanBlockedByUsers, Expire: time.Hour})

	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "temporary_ban")))
	assert.Equal(t, zapcore.ErrorLevel, logs.FilterMessage("whatsapp temporary ban").All()[0].Level)
}

func TestHandleEvent_KeepAliveTimeout_Counts(t *testing.T) {
	p, logs := newObservedProvider(t)
	tenant := "tenant-keepalive"

	p.handleEvent(tenant, &events.KeepAliveTimeout{})

	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "keepalive_timeout")))
	assert.Equal(t, 1, logs.FilterMessage("whatsapp keepalive timeout").Len())
}

func TestHandleEvent_LoggedOut_RemovesClientAndCounts(t *testing.T) {
	p, _ := newObservedProvider(t)
	tenant := "tenant-loggedout"
	connectionUp.WithLabelValues(tenant).Set(1)
	p.clients[tenant] = nil // presence is what matters; LoggedOut must remove it

	p.handleEvent(tenant, &events.LoggedOut{})

	p.mu.RLock()
	_, present := p.clients[tenant]
	p.mu.RUnlock()
	assert.False(t, present, "LoggedOut must remove the client from the map")
	assert.Equal(t, 0.0, testutil.ToFloat64(connectionUp.WithLabelValues(tenant)))
	assert.Equal(t, 1.0, testutil.ToFloat64(disconnectTotal.WithLabelValues(tenant, "logged_out")))
}
