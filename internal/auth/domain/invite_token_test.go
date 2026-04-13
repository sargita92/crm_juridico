package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInviteToken_Success(t *testing.T) {
	expires := time.Now().Add(24 * time.Hour)
	token, err := NewInviteToken("id-1", "tenant-1", "user-1", []string{"group-1"}, expires)
	require.NoError(t, err)
	assert.Equal(t, "id-1", token.ID)
	assert.Equal(t, "tenant-1", token.TenantID)
	assert.Equal(t, "user-1", token.CreatedBy)
	assert.Equal(t, []string{"group-1"}, token.GroupIDs)
	assert.Len(t, token.Token, 64) // 32 bytes hex
	assert.Nil(t, token.UsedAt)
}

func TestNewInviteToken_EmptyTenantID(t *testing.T) {
	_, err := NewInviteToken("id", "", "user-1", []string{"g"}, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewInviteToken_EmptyCreatedBy(t *testing.T) {
	_, err := NewInviteToken("id", "tenant", "", []string{"g"}, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrUserIDRequired)
}

func TestNewInviteToken_NoGroups(t *testing.T) {
	_, err := NewInviteToken("id", "tenant", "user", nil, time.Now().Add(time.Hour))
	assert.ErrorIs(t, err, ErrGroupIDRequired)
}

func TestInviteToken_IsExpired(t *testing.T) {
	past := &InviteToken{ExpiresAt: time.Now().Add(-time.Hour)}
	assert.True(t, past.IsExpired())

	future := &InviteToken{ExpiresAt: time.Now().Add(time.Hour)}
	assert.False(t, future.IsExpired())
}

func TestInviteToken_IsUsed(t *testing.T) {
	token := &InviteToken{}
	assert.False(t, token.IsUsed())

	now := time.Now()
	token.UsedAt = &now
	assert.True(t, token.IsUsed())
}

func TestInviteToken_MarkUsed(t *testing.T) {
	token := &InviteToken{}
	token.MarkUsed("user-99")
	assert.True(t, token.IsUsed())
	assert.Equal(t, "user-99", token.UsedBy)
	assert.NotNil(t, token.UsedAt)
}

func TestInviteToken_Validate_Valid(t *testing.T) {
	token := &InviteToken{ExpiresAt: time.Now().Add(time.Hour)}
	assert.NoError(t, token.Validate())
}

func TestInviteToken_Validate_Used(t *testing.T) {
	now := time.Now()
	token := &InviteToken{
		UsedAt:    &now,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	assert.ErrorIs(t, token.Validate(), ErrInviteTokenUsed)
}

func TestInviteToken_Validate_Expired(t *testing.T) {
	token := &InviteToken{ExpiresAt: time.Now().Add(-time.Hour)}
	assert.ErrorIs(t, token.Validate(), ErrInviteTokenExpired)
}
