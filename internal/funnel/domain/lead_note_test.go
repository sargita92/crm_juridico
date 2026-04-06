package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLeadNote_Success(t *testing.T) {
	note, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "Ligar amanha", "user-1")
	require.NoError(t, err)
	assert.Equal(t, "lead-1", note.LeadID)
	assert.Equal(t, "tenant-1", note.TenantID)
	assert.Equal(t, "Ligar amanha", note.Content)
	assert.Equal(t, "user-1", note.CreatedBy)
	assert.False(t, note.CreatedAt.IsZero())
}

func TestNewLeadNote_ContentRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "", "user-1")
	assert.ErrorIs(t, err, ErrNoteContentRequired)
}

func TestNewLeadNote_ContentTooLong(t *testing.T) {
	long := make([]byte, 2001)
	for i := range long {
		long[i] = 'a'
	}
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", string(long), "user-1")
	assert.ErrorIs(t, err, ErrNoteContentTooLong)
}

func TestNewLeadNote_CreatedByRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "tenant-1", "note", "")
	assert.ErrorIs(t, err, ErrNoteCreatedByRequired)
}

func TestNewLeadNote_LeadIDRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "", "tenant-1", "note", "user-1")
	assert.ErrorIs(t, err, ErrLeadNotFound)
}

func TestNewLeadNote_TenantIDRequired(t *testing.T) {
	_, err := NewLeadNote(uuid.New().String(), "lead-1", "", "note", "user-1")
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}
