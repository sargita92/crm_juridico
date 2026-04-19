package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type recordingFileStorer struct {
	inbound []domain.InboundMediaInput
	err     error
}

func (r *recordingFileStorer) StoreInbound(_ context.Context, in domain.InboundMediaInput) (string, error) {
	r.inbound = append(r.inbound, in)
	if r.err != nil {
		return "", r.err
	}
	return "file-" + in.MessageID, nil
}

func (r *recordingFileStorer) StoreOutbound(_ context.Context, _ domain.OutboundMediaInput) (string, error) {
	return "", nil
}

func TestReceiveMessage_Media_CallsFileStorerAndPersistsMediaType(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()
	fs := &recordingFileStorer{}
	uc.SetFileStorer(fs)

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "t1", SenderJID: "1@s.whatsapp.net",
		SenderName: "U", SenderPhone: "+1",
		Content:       "",
		WhatsAppMsgID: "wa-1",
		Timestamp:     time.Now(),
		Media: &domain.MediaPayload{
			Type:     domain.MessageTypeDocument,
			Name:     "contrato.pdf",
			MimeType: "application/pdf",
			Content:  []byte("bytes"),
		},
	})
	require.NoError(t, err)

	// Message persisted with document type, empty content accepted.
	require.Len(t, msgRepo.messages, 1)
	var persisted *domain.Message
	for _, m := range msgRepo.messages {
		persisted = m
	}
	require.NotNil(t, persisted)
	assert.Equal(t, domain.MessageTypeDocument, persisted.Type)

	// FileStorer was invoked with the saved message id and bytes.
	require.Len(t, fs.inbound, 1)
	assert.Equal(t, "t1", fs.inbound[0].TenantID)
	assert.Equal(t, persisted.ID, fs.inbound[0].MessageID)
	assert.Equal(t, "contrato.pdf", fs.inbound[0].Name)
	assert.Equal(t, "application/pdf", fs.inbound[0].MimeType)
	assert.Equal(t, []byte("bytes"), fs.inbound[0].Content)
}

func TestReceiveMessage_Media_StorerErrorDoesNotFailIngestion(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()
	fs := &recordingFileStorer{err: errors.New("disk full")}
	uc.SetFileStorer(fs)

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "t1", SenderJID: "1@s.whatsapp.net",
		SenderName: "U", SenderPhone: "+1",
		Content:       "caption",
		WhatsAppMsgID: "wa-1",
		Timestamp:     time.Now(),
		Media: &domain.MediaPayload{
			Type: domain.MessageTypeImage, Name: "img.jpg",
			MimeType: "image/jpeg", Content: []byte("x"),
		},
	})
	require.NoError(t, err, "ingestion must not fail on storage error")

	// Message was still persisted.
	require.Len(t, msgRepo.messages, 1)
}

func TestReceiveMessage_Media_NilStorerDoesNotPanic(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()
	// fileStorer intentionally left nil

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "t1", SenderJID: "1@s.whatsapp.net",
		SenderName: "U", SenderPhone: "+1",
		Content: "", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
		Media: &domain.MediaPayload{
			Type: domain.MessageTypeAudio, Name: "a.ogg",
			MimeType: "audio/ogg", Content: []byte("x"),
		},
	})
	require.NoError(t, err)
	require.Len(t, msgRepo.messages, 1)
}

func TestReceiveMessage_MediaWithoutTypeFallsBackToOther(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()
	fs := &recordingFileStorer{}
	uc.SetFileStorer(fs)

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "t1", SenderJID: "1@s.whatsapp.net",
		SenderName: "U", SenderPhone: "+1",
		Content: "", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
		Media: &domain.MediaPayload{
			Name: "blob", MimeType: "application/x-tar", Content: []byte("x"),
		},
	})
	require.NoError(t, err)

	var persisted *domain.Message
	for _, m := range msgRepo.messages {
		persisted = m
	}
	require.NotNil(t, persisted)
	assert.Equal(t, domain.MessageTypeOther, persisted.Type)
}

func TestReceiveMessage_EmptyContentNoMedia_Discarded(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()
	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "t1", SenderJID: "1@s.whatsapp.net",
		SenderName: "U", SenderPhone: "+1",
		Content: "", WhatsAppMsgID: "wa-x", Timestamp: time.Now(),
		Media: nil,
	})
	require.NoError(t, err)
	assert.Empty(t, msgRepo.messages, "empty message must be silently discarded")
}
