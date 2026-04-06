package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectWhatsApp_Success(t *testing.T) {
	provider := newMockProvider()
	uc := NewConnectWhatsAppUseCase(provider)

	qrChan, err := uc.Execute(context.Background(), "tenant-1")

	require.NoError(t, err)
	qr := <-qrChan
	assert.Equal(t, "qr-code-data", qr)
	assert.True(t, provider.IsConnected("tenant-1"))
}

func TestConnectWhatsApp_ProviderError(t *testing.T) {
	provider := newMockProvider()
	provider.connectErr = errors.New("connection failed")
	uc := NewConnectWhatsAppUseCase(provider)

	_, err := uc.Execute(context.Background(), "tenant-1")

	assert.Error(t, err)
}
