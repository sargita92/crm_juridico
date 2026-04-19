package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
	waDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func newAdapter() (*WhatsAppFileAdapter, *mockFileRepo) {
	repo := newMockFileRepo()
	storage := newMockStorage()
	lookup := newMockLeadLookup()
	uc := NewStoreFileUseCase(repo, storage, lookup, 0)
	return NewWhatsAppFileAdapter(uc), repo
}

func TestWhatsAppAdapter_StoreInboundRecordsFile(t *testing.T) {
	a, repo := newAdapter()
	id, err := a.StoreInbound(context.Background(), waDomain.InboundMediaInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		MessageID:      "m1",
		Name:           "doc.pdf",
		MimeType:       "application/pdf",
		Content:        []byte("bytes"),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	saved, _ := repo.FindByID(context.Background(), "t1", id)
	require.NotNil(t, saved)
	assert.Equal(t, domain.DirectionInbound, saved.Direction)
	require.NotNil(t, saved.MessageID)
	assert.Equal(t, "m1", *saved.MessageID)
}

func TestWhatsAppAdapter_StoreOutboundRecordsFile(t *testing.T) {
	a, repo := newAdapter()
	id, err := a.StoreOutbound(context.Background(), waDomain.OutboundMediaInput{
		TenantID:       "t1",
		ConversationID: "c1",
		ContactID:      "k1",
		MessageID:      "m1",
		Name:           "audio.ogg",
		MimeType:       "audio/ogg",
		Content:        []byte("bytes"),
	})
	require.NoError(t, err)
	saved, _ := repo.FindByID(context.Background(), "t1", id)
	require.NotNil(t, saved)
	assert.Equal(t, domain.DirectionOutbound, saved.Direction)
	assert.Equal(t, domain.MediaTypeAudio, saved.MediaType)
}

func TestWhatsAppAdapter_EmptyMessageIDStoredAsNil(t *testing.T) {
	a, repo := newAdapter()
	id, err := a.StoreInbound(context.Background(), waDomain.InboundMediaInput{
		TenantID: "t1", ConversationID: "c1", ContactID: "k1",
		MessageID: "",
		Name:      "x.pdf", MimeType: "application/pdf", Content: []byte("x"),
	})
	require.NoError(t, err)
	saved, _ := repo.FindByID(context.Background(), "t1", id)
	require.NotNil(t, saved)
	assert.Nil(t, saved.MessageID)
}
