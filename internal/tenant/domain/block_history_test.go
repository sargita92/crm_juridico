package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockHistoryEntry_BlockSuccess(t *testing.T) {
	entry, err := NewBlockHistoryEntry("id-1", "tenant-1", BlockActionBlock, "Inadimplencia", "admin-1")
	require.NoError(t, err)
	assert.Equal(t, "id-1", entry.ID)
	assert.Equal(t, "tenant-1", entry.TenantID)
	assert.Equal(t, BlockActionBlock, entry.Action)
	assert.Equal(t, "Inadimplencia", entry.Reason)
	assert.Equal(t, "admin-1", entry.PerformedBy)
	assert.False(t, entry.CreatedAt.IsZero())
}

func TestNewBlockHistoryEntry_UnblockSuccess(t *testing.T) {
	entry, err := NewBlockHistoryEntry("id-2", "tenant-1", BlockActionUnblock, "Regularizado", "admin-1")
	require.NoError(t, err)
	assert.Equal(t, BlockActionUnblock, entry.Action)
}

func TestNewBlockHistoryEntry_EmptyTenantID(t *testing.T) {
	_, err := NewBlockHistoryEntry("id-1", "", BlockActionBlock, "Motivo", "admin-1")
	assert.ErrorIs(t, err, ErrHistoryTenantIDRequired)
}

func TestNewBlockHistoryEntry_EmptyReason(t *testing.T) {
	_, err := NewBlockHistoryEntry("id-1", "tenant-1", BlockActionBlock, "", "admin-1")
	assert.ErrorIs(t, err, ErrBlockReasonRequired)
}

func TestNewBlockHistoryEntry_ReasonTooLong(t *testing.T) {
	longReason := make([]byte, 501)
	for i := range longReason {
		longReason[i] = 'a'
	}
	_, err := NewBlockHistoryEntry("id-1", "tenant-1", BlockActionBlock, string(longReason), "admin-1")
	assert.ErrorIs(t, err, ErrReasonTooLong)
}

func TestNewBlockHistoryEntry_EmptyPerformedBy(t *testing.T) {
	_, err := NewBlockHistoryEntry("id-1", "tenant-1", BlockActionBlock, "Motivo", "")
	assert.ErrorIs(t, err, ErrHistoryPerformedByRequired)
}

func TestNewBlockHistoryEntry_InvalidAction(t *testing.T) {
	_, err := NewBlockHistoryEntry("id-1", "tenant-1", BlockAction("invalid"), "Motivo", "admin-1")
	assert.ErrorIs(t, err, ErrInvalidBlockAction)
}
