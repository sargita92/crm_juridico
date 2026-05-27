package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestZapWaLogger_MapsLevelsAndFormats(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	wl := newWaLogger(zap.New(core), "Client")

	wl.Errorf("boom %d", 1)
	wl.Warnf("careful %s", "now")
	wl.Infof("info %d", 2)
	wl.Debugf("debug %d", 3)

	entries := logs.All()
	assert.Len(t, entries, 4)

	assert.Equal(t, zapcore.ErrorLevel, entries[0].Level)
	assert.Equal(t, "boom 1", entries[0].Message)
	assert.Equal(t, zapcore.WarnLevel, entries[1].Level)
	assert.Equal(t, "careful now", entries[1].Message)
	assert.Equal(t, zapcore.InfoLevel, entries[2].Level)
	assert.Equal(t, "info 2", entries[2].Message)
	assert.Equal(t, zapcore.DebugLevel, entries[3].Level)
	assert.Equal(t, "debug 3", entries[3].Message)

	// every entry carries the whatsmeow module for filtering
	for _, e := range entries {
		assert.Equal(t, "Client", e.ContextMap()["wa_module"])
	}
}

func TestZapWaLogger_SubComposesModule(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	wl := newWaLogger(zap.New(core), "Client")

	wl.Sub("Socket").Infof("connected")

	entries := logs.All()
	assert.Len(t, entries, 1)
	assert.Equal(t, "Client/Socket", entries[0].ContextMap()["wa_module"])
}
