package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// stubMsgRepoForClear is a minimal whatsappDomain.MessageRepository used to
// verify PlaygroundMessageAdapter.ClearHistory delegates correctly.
type stubMsgRepoForClear struct {
	deletedConvID string
	count         int64
	err           error
}

func (s *stubMsgRepoForClear) Create(context.Context, *whatsappDomain.Message) error { return nil }
func (s *stubMsgRepoForClear) FindByConversationID(context.Context, string, whatsappDomain.MessageFilter) ([]whatsappDomain.Message, error) {
	return nil, nil
}
func (s *stubMsgRepoForClear) FindByWhatsAppMsgID(context.Context, string) (*whatsappDomain.Message, error) {
	return nil, whatsappDomain.ErrMessageNotFound
}
func (s *stubMsgRepoForClear) Update(context.Context, *whatsappDomain.Message) error { return nil }
func (s *stubMsgRepoForClear) DeleteByConversationID(_ context.Context, conversationID string) (int64, error) {
	s.deletedConvID = conversationID
	return s.count, s.err
}

func TestPlaygroundMessageAdapter_ClearHistory_DelegatesToRepo(t *testing.T) {
	repo := &stubMsgRepoForClear{count: 7}
	adapter := NewPlaygroundMessageAdapter(repo)

	deleted, err := adapter.ClearHistory(context.Background(), "conv-1")
	require.NoError(t, err)
	assert.Equal(t, int64(7), deleted)
	assert.Equal(t, "conv-1", repo.deletedConvID, "should delete the given conversation")
}

func TestPlaygroundMessageAdapter_ClearHistory_WrapsError(t *testing.T) {
	repo := &stubMsgRepoForClear{err: errors.New("boom")}
	adapter := NewPlaygroundMessageAdapter(repo)

	_, err := adapter.ClearHistory(context.Background(), "conv-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "playground_message_adapter")
}
