package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConnectionStatus_Connected(t *testing.T) {
	provider := newMockProvider()
	provider.connected["tenant-1"] = true
	uc := NewGetConnectionStatusUseCase(provider)

	status, err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	assert.True(t, status.Connected)
}

func TestGetConnectionStatus_Disconnected(t *testing.T) {
	provider := newMockProvider()
	uc := NewGetConnectionStatusUseCase(provider)

	status, err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	assert.False(t, status.Connected)
}
