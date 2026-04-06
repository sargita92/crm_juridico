package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisconnectWhatsApp_Success(t *testing.T) {
	provider := newMockProvider()
	provider.connected["tenant-1"] = true
	uc := NewDisconnectWhatsAppUseCase(provider)

	err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	assert.False(t, provider.IsConnected("tenant-1"))
}
