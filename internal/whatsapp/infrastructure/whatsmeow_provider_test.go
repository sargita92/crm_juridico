package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func TestWhatsmeowProvider_ImplementsInterface(t *testing.T) {
	// Compile-time check is done via var _ in the source file.
	// This test verifies the constructor works.
	log, _ := zap.NewDevelopment()
	provider := NewWhatsmeowProvider(t.TempDir(), log)

	assert.NotNil(t, provider)
	assert.False(t, provider.IsConnected("nonexistent-tenant"))
}

func TestWhatsmeowProvider_SetMessageHandler(t *testing.T) {
	log, _ := zap.NewDevelopment()
	provider := NewWhatsmeowProvider(t.TempDir(), log)

	provider.SetMessageHandler(func(ctx context.Context, event domain.IncomingMessage) {})

	assert.NotNil(t, provider.handler)
}

func TestWhatsmeowProvider_IsConnected_NoClient(t *testing.T) {
	log, _ := zap.NewDevelopment()
	provider := NewWhatsmeowProvider(t.TempDir(), log)

	assert.False(t, provider.IsConnected("tenant-1"))
}

func TestWhatsmeowProvider_Disconnect_NoClient_NoPanic(t *testing.T) {
	log, _ := zap.NewDevelopment()
	provider := NewWhatsmeowProvider(t.TempDir(), log)

	assert.NotPanics(t, func() {
		_ = provider.Disconnect(context.Background(), "nonexistent")
	})
}

func TestWhatsmeowProvider_Shutdown_Empty_NoPanic(t *testing.T) {
	log, _ := zap.NewDevelopment()
	provider := NewWhatsmeowProvider(t.TempDir(), log)

	assert.NotPanics(t, func() {
		provider.Shutdown()
	})
}

func TestSanitizeTenantID(t *testing.T) {
	assert.Equal(t, "abc123def456", sanitizeTenantID("abc-123-def-456"))
	assert.Equal(t, "nohyphens", sanitizeTenantID("nohyphens"))
}

func TestRecipientJID_StripsDevicePart(t *testing.T) {
	sender := types.JID{User: "553184019118", Device: 56, Server: types.DefaultUserServer}

	assert.Equal(t, "553184019118@s.whatsapp.net", recipientJID(sender))
}

func TestRecipientJID_NoDevice(t *testing.T) {
	sender := types.JID{User: "553184019118", Server: types.DefaultUserServer}

	assert.Equal(t, "553184019118@s.whatsapp.net", recipientJID(sender))
}

func TestIsDirectMessage(t *testing.T) {
	cases := []struct {
		name   string
		server string
		want   bool
	}{
		{"dm user", types.DefaultUserServer, true},
		{"group", types.GroupServer, false},
		{"newsletter", types.NewsletterServer, false},
		{"broadcast", types.BroadcastServer, false},
		{"hidden lid", types.HiddenUserServer, false},
		{"empty", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, isDirectMessage(types.JID{Server: c.server}))
		})
	}
}
