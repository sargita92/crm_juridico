package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func TestResolvePeriod_Presets(t *testing.T) {
	tests := []struct {
		preset string
		from   bool
	}{
		{"today", true},
		{"7d", true},
		{"30d", true},
	}
	for _, c := range tests {
		t.Run(c.preset, func(t *testing.T) {
			f, _, err := resolvePeriod(c.preset, "", "")
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})
	}
}

func TestResolvePeriod_Custom_InvalidFrom(t *testing.T) {
	_, _, err := resolvePeriod("custom", "abc", "")
	assert.Error(t, err)
}

func TestResolvePeriod_Custom_InvalidTo(t *testing.T) {
	_, _, err := resolvePeriod("custom", "", "abc")
	assert.Error(t, err)
}

func TestResolvePeriod_Custom_FromAfterTo(t *testing.T) {
	_, _, err := resolvePeriod("custom", "2026-04-10", "2026-04-01")
	assert.Error(t, err)
}

func TestResolvePeriod_All_ReturnsNil(t *testing.T) {
	f, to, err := resolvePeriod("all", "", "")
	assert.NoError(t, err)
	assert.Nil(t, f)
	assert.Nil(t, to)
}

func TestResolvePeriod_Unknown(t *testing.T) {
	_, _, err := resolvePeriod("lifetime", "", "")
	assert.Error(t, err)
}

func TestDirectionLabel(t *testing.T) {
	assert.Equal(t, "recebido", directionLabel(domain.DirectionInbound))
	assert.Equal(t, "enviado", directionLabel(domain.DirectionOutbound))
	assert.Equal(t, "unknown", directionLabel(domain.Direction("unknown")))
}

func TestMediaIcon(t *testing.T) {
	assert.NotEmpty(t, mediaIcon(domain.MediaTypeImage))
	assert.NotEmpty(t, mediaIcon(domain.MediaTypeDocument))
	assert.NotEmpty(t, mediaIcon(domain.MediaTypeAudio))
	assert.NotEmpty(t, mediaIcon(domain.MediaTypeVideo))
	assert.NotEmpty(t, mediaIcon(domain.MediaTypeOther))
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()
	assert.Equal(t, "agora", relativeTime(now.Add(-10*time.Second)))
	assert.Equal(t, "5m", relativeTime(now.Add(-5*time.Minute)))
	assert.Equal(t, "2h", relativeTime(now.Add(-2*time.Hour)))
	assert.Equal(t, "3d", relativeTime(now.Add(-3*24*time.Hour)))
	// 60d: should format date
	assert.Regexp(t, `\d{2}/\d{2}/\d{4}`, relativeTime(now.Add(-60*24*time.Hour)))
}

func TestFormatInt(t *testing.T) {
	assert.Equal(t, "0", formatInt(0))
	assert.Equal(t, "7", formatInt(7))
	assert.Equal(t, "42", formatInt(42))
	assert.Equal(t, "1000", formatInt(1000))
	assert.Equal(t, "0", formatInt(-3), "negative clamped to 0")
}

func TestIsNotFoundErr(t *testing.T) {
	assert.True(t, isNotFoundErr(domain.ErrFileNotFound))
	assert.False(t, isNotFoundErr(domain.ErrTenantIDRequired))
}

func TestStatusFor(t *testing.T) {
	h := &Handler{}
	assert.Equal(t, 200, h.statusFor(nil))
	assert.Equal(t, 404, h.statusFor(domain.ErrFileNotFound))
	assert.Equal(t, 500, h.statusFor(domain.ErrTenantIDRequired))
}

func TestNewHandler_NilLogDoesNotPanic(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil)
	assert.NotNil(t, h)
}
