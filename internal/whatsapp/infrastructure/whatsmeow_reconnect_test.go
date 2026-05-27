package infrastructure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.uber.org/zap"
)

// createUnpairedStore creates a whatsmeow SQLite store file for tenantID with a
// device that was never paired (Store.ID == nil), mirroring what Connect leaves
// behind after an abandoned QR flow.
func createUnpairedStore(t *testing.T, storeDir, tenantID string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(storeDir, 0750))
	dbPath := filepath.Join(storeDir, sanitizeTenantID(tenantID)+".db")
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath)
	container, err := sqlstore.New(context.Background(), "sqlite3", dsn, nil)
	require.NoError(t, err)
	require.NoError(t, container.Upgrade(context.Background()))
	_, err = container.GetFirstDevice(context.Background())
	require.NoError(t, err)
	require.NoError(t, container.Close())
}

func TestHasPairedSession_NoFile_ReturnsFalseAndCreatesNothing(t *testing.T) {
	dir := t.TempDir()
	p := NewWhatsmeowProvider(dir, zap.NewNop())

	paired, err := p.hasPairedSession(context.Background(), "tenant-without-session")

	assert.NoError(t, err)
	assert.False(t, paired)
	// the check must not create a store file as a side effect
	_, statErr := os.Stat(filepath.Join(dir, "tenantwithoutsession.db"))
	assert.True(t, os.IsNotExist(statErr), "hasPairedSession must not create a db file when none exists")
}

func TestHasPairedSession_UnpairedDevice_ReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	p := NewWhatsmeowProvider(dir, zap.NewNop())
	createUnpairedStore(t, dir, "tenant-unpaired")

	paired, err := p.hasPairedSession(context.Background(), "tenant-unpaired")

	assert.NoError(t, err)
	assert.False(t, paired, "an unpaired device (Store.ID nil) must not count as a session")
}

func TestReconnectExisting_NoSessions_DoesNotConnect(t *testing.T) {
	dir := t.TempDir()
	p := NewWhatsmeowProvider(dir, zap.NewNop())

	assert.NotPanics(t, func() {
		p.ReconnectExisting(context.Background(), []string{"t1", "t2"})
	})

	assert.False(t, p.IsConnected("t1"))
	assert.False(t, p.IsConnected("t2"))
	p.mu.RLock()
	clientCount := len(p.clients)
	p.mu.RUnlock()
	assert.Equal(t, 0, clientCount, "no clients should be created for tenants without a paired session")
}
