package infrastructure

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// --- UserNameAdapter ---

type stubUserRepo struct {
	user *authdomain.User
	err  error
}

func (s *stubUserRepo) Create(context.Context, *authdomain.User) error { return nil }
func (s *stubUserRepo) Update(context.Context, *authdomain.User) error { return nil }
func (s *stubUserRepo) FindByID(_ context.Context, id string) (*authdomain.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}
func (s *stubUserRepo) FindByEmail(context.Context, string) (*authdomain.User, error) {
	return nil, errors.New("not used")
}
func (s *stubUserRepo) ExistsByEmail(context.Context, string) (bool, error) { return false, nil }

func TestUserNameAdapter_FindNameByID_Success(t *testing.T) {
	repo := &stubUserRepo{user: &authdomain.User{ID: "u1", Name: "Ana Silva"}}
	a := NewUserNameAdapter(repo)
	name, err := a.FindNameByID(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, "Ana Silva", name)
}

func TestUserNameAdapter_FindNameByID_Error(t *testing.T) {
	repo := &stubUserRepo{err: errors.New("boom")}
	a := NewUserNameAdapter(repo)
	_, err := a.FindNameByID(context.Background(), "u1")
	assert.Error(t, err)
}

// --- WhatsAppContactAdapter ---

type stubContactRepo struct {
	contact *whatsappdomain.Contact
	err     error
}

func (s *stubContactRepo) Create(context.Context, *whatsappdomain.Contact) error { return nil }
func (s *stubContactRepo) FindByID(_ context.Context, id string) (*whatsappdomain.Contact, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.contact, nil
}
func (s *stubContactRepo) FindByWhatsAppID(context.Context, string, string) (*whatsappdomain.Contact, error) {
	return nil, errors.New("not used")
}
func (s *stubContactRepo) Update(context.Context, *whatsappdomain.Contact) error { return nil }

func TestWhatsAppContactAdapter_FindByID_Success(t *testing.T) {
	repo := &stubContactRepo{contact: &whatsappdomain.Contact{ID: "c1", Name: "Joao", Phone: "+5511999"}}
	a := NewWhatsAppContactAdapter(repo)
	info, err := a.FindByID(context.Background(), "c1")
	require.NoError(t, err)
	assert.Equal(t, "Joao", info.Name)
	assert.Equal(t, "+5511999", info.Phone)
}

func TestWhatsAppContactAdapter_FindByID_Error(t *testing.T) {
	repo := &stubContactRepo{err: errors.New("nope")}
	a := NewWhatsAppContactAdapter(repo)
	_, err := a.FindByID(context.Background(), "c1")
	assert.Error(t, err)
}

// --- WhatsAppMessageAdapter ---

type stubMessageRepo struct {
	messages []whatsappdomain.Message
	err      error
}

func (s *stubMessageRepo) Create(context.Context, *whatsappdomain.Message) error { return nil }
func (s *stubMessageRepo) FindByConversationID(_ context.Context, _ string, _ whatsappdomain.MessageFilter) ([]whatsappdomain.Message, error) {
	return s.messages, s.err
}
func (s *stubMessageRepo) FindByWhatsAppMsgID(context.Context, string) (*whatsappdomain.Message, error) {
	return nil, errors.New("not used")
}
func (s *stubMessageRepo) Update(context.Context, *whatsappdomain.Message) error { return nil }
func (s *stubMessageRepo) DeleteByConversationID(context.Context, string) (int64, error) {
	return 0, nil
}

func TestWhatsAppMessageAdapter_FindRecent_Success(t *testing.T) {
	now := time.Now()
	repo := &stubMessageRepo{
		messages: []whatsappdomain.Message{
			{Direction: whatsappdomain.MessageDirectionIncoming, Content: "oi", Timestamp: now},
			{Direction: whatsappdomain.MessageDirectionOutgoing, Content: "ola", Timestamp: now},
		},
	}
	a := NewWhatsAppMessageAdapter(repo)
	out, err := a.FindRecentByConversationID(context.Background(), "conv-1", 10)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, "oi", out[0].Content)
	assert.Equal(t, string(whatsappdomain.MessageDirectionIncoming), out[0].Direction)
}

func TestWhatsAppMessageAdapter_FindRecent_Error(t *testing.T) {
	repo := &stubMessageRepo{err: errors.New("db err")}
	a := NewWhatsAppMessageAdapter(repo)
	_, err := a.FindRecentByConversationID(context.Background(), "conv-1", 10)
	assert.Error(t, err)
}
