package domain

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// InviteToken represents a one-time invitation link for joining a tenant with specific groups.
type InviteToken struct {
	ID        string
	TenantID  string
	Token     string
	CreatedBy string
	GroupIDs  []string
	ExpiresAt time.Time
	UsedAt    *time.Time
	UsedBy    string
	CreatedAt time.Time
}

// NewInviteToken creates a new InviteToken with a cryptographically random token.
func NewInviteToken(id, tenantID, createdBy string, groupIDs []string, expiresAt time.Time) (*InviteToken, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if createdBy == "" {
		return nil, ErrUserIDRequired
	}
	if len(groupIDs) == 0 {
		return nil, ErrGroupIDRequired
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	return &InviteToken{
		ID:        id,
		TenantID:  tenantID,
		Token:     hex.EncodeToString(tokenBytes),
		CreatedBy: createdBy,
		GroupIDs:  groupIDs,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

// IsExpired returns true if the token has passed its expiry time.
func (t *InviteToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *InviteToken) IsUsed() bool {
	return t.UsedAt != nil
}

// MarkUsed stamps the token as used by the given userID.
func (t *InviteToken) MarkUsed(userID string) {
	now := time.Now()
	t.UsedAt = &now
	t.UsedBy = userID
}

// Validate checks the token is still valid for redemption.
func (t *InviteToken) Validate() error {
	if t.IsUsed() {
		return ErrInviteTokenUsed
	}
	if t.IsExpired() {
		return ErrInviteTokenExpired
	}
	return nil
}
