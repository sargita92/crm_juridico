package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContact_WithValidData_ReturnsContact(t *testing.T) {
	contact, err := NewContact("uuid-1", "tenant-1", "Maria Silva", "+5511999990001", "5511999990001@s.whatsapp.net")

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", contact.ID)
	assert.Equal(t, "tenant-1", contact.TenantID)
	assert.Equal(t, "Maria Silva", contact.Name)
	assert.Equal(t, "+5511999990001", contact.Phone)
	assert.Equal(t, "5511999990001@s.whatsapp.net", contact.WhatsAppID)
	assert.False(t, contact.CreatedAt.IsZero())
	assert.False(t, contact.UpdatedAt.IsZero())
}

func TestNewContact_WithEmptyName_ReturnsContact(t *testing.T) {
	contact, err := NewContact("uuid-1", "tenant-1", "", "+5511999990001", "5511999990001@s.whatsapp.net")

	require.NoError(t, err)
	assert.Empty(t, contact.Name)
}

func TestNewContact_EmptyTenantID_ReturnsError(t *testing.T) {
	_, err := NewContact("uuid-1", "", "Maria", "+5511999990001", "jid")

	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewContact_EmptyPhone_ReturnsError(t *testing.T) {
	_, err := NewContact("uuid-1", "tenant-1", "Maria", "", "jid")

	assert.ErrorIs(t, err, ErrContactPhoneRequired)
}

func TestNewContact_EmptyWhatsAppID_ReturnsError(t *testing.T) {
	_, err := NewContact("uuid-1", "tenant-1", "Maria", "+5511999990001", "")

	assert.ErrorIs(t, err, ErrContactWhatsAppIDRequired)
}

func TestNewContact_NameTooLong_ReturnsError(t *testing.T) {
	longName := strings.Repeat("a", MaxContactNameLength+1)
	_, err := NewContact("uuid-1", "tenant-1", longName, "+5511999990001", "jid")

	assert.ErrorIs(t, err, ErrContactNameTooLong)
}

func TestNewContact_NameAtMaxLength_ReturnsContact(t *testing.T) {
	maxName := strings.Repeat("a", MaxContactNameLength)
	contact, err := NewContact("uuid-1", "tenant-1", maxName, "+5511999990001", "jid")

	require.NoError(t, err)
	assert.Len(t, contact.Name, MaxContactNameLength)
}

func TestContact_UpdateName_ChangesName(t *testing.T) {
	contact, _ := NewContact("uuid-1", "tenant-1", "Maria", "+5511999990001", "jid")
	oldUpdatedAt := contact.UpdatedAt

	contact.UpdateName("Maria Silva")

	assert.Equal(t, "Maria Silva", contact.Name)
	assert.True(t, contact.UpdatedAt.After(oldUpdatedAt) || contact.UpdatedAt.Equal(oldUpdatedAt))
}

func TestContact_UpdateName_EmptyName_NoChange(t *testing.T) {
	contact, _ := NewContact("uuid-1", "tenant-1", "Maria", "+5511999990001", "jid")

	contact.UpdateName("")

	assert.Equal(t, "Maria", contact.Name)
}

func TestContact_UpdateName_SameName_NoChange(t *testing.T) {
	contact, _ := NewContact("uuid-1", "tenant-1", "Maria", "+5511999990001", "jid")
	originalUpdatedAt := contact.UpdatedAt

	contact.UpdateName("Maria")

	assert.Equal(t, "Maria", contact.Name)
	assert.Equal(t, originalUpdatedAt, contact.UpdatedAt)
}
