package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConversation_WithValidData_ReturnsConversation(t *testing.T) {
	conv, err := NewConversation("uuid-1", "tenant-1", "contact-1")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", conv.ID)
	assert.Equal(t, "tenant-1", conv.TenantID)
	assert.Equal(t, "contact-1", conv.ContactID)
	assert.Equal(t, ConversationStatusOpen, conv.Status)
	assert.Equal(t, 0, conv.UnreadCount)
	assert.False(t, conv.LastMessageAt.IsZero())
	assert.False(t, conv.CreatedAt.IsZero())
	assert.False(t, conv.UpdatedAt.IsZero())
}

func TestNewConversation_EmptyTenantID_ReturnsError(t *testing.T) {
	_, err := NewConversation("uuid-1", "", "contact-1")

	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewConversation_EmptyContactID_ReturnsError(t *testing.T) {
	_, err := NewConversation("uuid-1", "tenant-1", "")

	assert.ErrorIs(t, err, ErrContactIDRequired)
}

func TestConversation_Close(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")

	conv.Close()

	assert.Equal(t, ConversationStatusClosed, conv.Status)
}

func TestConversation_RecordMessage_Incoming_IncrementsUnread(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")
	ts := time.Now().Add(5 * time.Minute)

	conv.RecordMessage(ts, true)

	assert.Equal(t, 1, conv.UnreadCount)
	assert.Equal(t, ts, conv.LastMessageAt)
}

func TestConversation_RecordMessage_Outgoing_NoUnreadChange(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")
	ts := time.Now().Add(5 * time.Minute)

	conv.RecordMessage(ts, false)

	assert.Equal(t, 0, conv.UnreadCount)
	assert.Equal(t, ts, conv.LastMessageAt)
}

func TestConversation_RecordMessage_MultipleIncoming_AccumulatesUnread(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")

	conv.RecordMessage(time.Now(), true)
	conv.RecordMessage(time.Now(), true)
	conv.RecordMessage(time.Now(), true)

	assert.Equal(t, 3, conv.UnreadCount)
}

func TestConversation_MarkRead(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")
	conv.RecordMessage(time.Now(), true)
	conv.RecordMessage(time.Now(), true)
	assert.Equal(t, 2, conv.UnreadCount)

	conv.MarkRead()

	assert.Equal(t, 0, conv.UnreadCount)
}

func TestConversation_IsOpen(t *testing.T) {
	conv, _ := NewConversation("uuid-1", "tenant-1", "contact-1")
	assert.True(t, conv.IsOpen())

	conv.Close()
	assert.False(t, conv.IsOpen())
}
