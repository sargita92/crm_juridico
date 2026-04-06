package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage_IncomingWithValidData_ReturnsMessage(t *testing.T) {
	ts := time.Now()
	msg, err := NewMessage("uuid-1", "conv-1", MessageDirectionIncoming, "Ola, preciso de ajuda", MessageTypeText, "wa-msg-1", ts)

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", msg.ID)
	assert.Equal(t, "conv-1", msg.ConversationID)
	assert.Equal(t, MessageDirectionIncoming, msg.Direction)
	assert.Equal(t, "Ola, preciso de ajuda", msg.Content)
	assert.Equal(t, MessageTypeText, msg.Type)
	assert.Equal(t, MessageStatusSent, msg.Status)
	assert.Equal(t, "wa-msg-1", msg.WhatsAppMsgID)
	assert.Equal(t, ts, msg.Timestamp)
	assert.False(t, msg.CreatedAt.IsZero())
}

func TestNewMessage_OutgoingWithValidData_StatusPending(t *testing.T) {
	msg, err := NewMessage("uuid-1", "conv-1", MessageDirectionOutgoing, "Ola!", MessageTypeText, "", time.Now())

	require.NoError(t, err)
	assert.Equal(t, MessageStatusPending, msg.Status)
}

func TestNewMessage_EmptyConversationID_ReturnsError(t *testing.T) {
	_, err := NewMessage("uuid-1", "", MessageDirectionIncoming, "Ola", MessageTypeText, "wa-1", time.Now())

	assert.ErrorIs(t, err, ErrConversationIDRequired)
}

func TestNewMessage_EmptyContent_ReturnsError(t *testing.T) {
	_, err := NewMessage("uuid-1", "conv-1", MessageDirectionIncoming, "", MessageTypeText, "wa-1", time.Now())

	assert.ErrorIs(t, err, ErrMessageContentRequired)
}

func TestNewMessage_ContentTooLong_ReturnsError(t *testing.T) {
	longContent := strings.Repeat("a", MaxMessageContentLength+1)
	_, err := NewMessage("uuid-1", "conv-1", MessageDirectionIncoming, longContent, MessageTypeText, "wa-1", time.Now())

	assert.ErrorIs(t, err, ErrMessageContentTooLong)
}

func TestNewMessage_ContentAtMaxLength_ReturnsMessage(t *testing.T) {
	maxContent := strings.Repeat("a", MaxMessageContentLength)
	msg, err := NewMessage("uuid-1", "conv-1", MessageDirectionIncoming, maxContent, MessageTypeText, "wa-1", time.Now())

	require.NoError(t, err)
	assert.Len(t, msg.Content, MaxMessageContentLength)
}

func TestNewMessage_EmptyWhatsAppMsgID_Allowed(t *testing.T) {
	msg, err := NewMessage("uuid-1", "conv-1", MessageDirectionOutgoing, "Ola", MessageTypeText, "", time.Now())

	require.NoError(t, err)
	assert.Empty(t, msg.WhatsAppMsgID)
}

func TestMessage_MarkSent(t *testing.T) {
	msg, _ := NewMessage("uuid-1", "conv-1", MessageDirectionOutgoing, "Ola", MessageTypeText, "", time.Now())
	assert.Equal(t, MessageStatusPending, msg.Status)

	msg.MarkSent()

	assert.Equal(t, MessageStatusSent, msg.Status)
}

func TestMessage_MarkFailed(t *testing.T) {
	msg, _ := NewMessage("uuid-1", "conv-1", MessageDirectionOutgoing, "Ola", MessageTypeText, "", time.Now())

	msg.MarkFailed()

	assert.Equal(t, MessageStatusFailed, msg.Status)
}
