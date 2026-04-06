package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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
		_ = provider.Disconnect(nil, "nonexistent")
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
