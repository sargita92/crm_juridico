package application

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	infra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
)

func setupInviteUC() (*InviteUserUseCase, *mockInviteTokenRepo, *mockUserRepo, *mockUserTenantRepo) {
	inviteRepo := newMockInviteTokenRepo()
	userRepo := newMockUserRepo()
	userTenantRepo := newMockUserTenantRepo()
	hasher := &mockHasher{}

	uc := NewInviteUserUseCase(inviteRepo, userRepo, userTenantRepo, hasher)
	return uc, inviteRepo, userRepo, userTenantRepo
}

func TestGenerateInvite_Success(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	before := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("sent"))
	out, err := uc.GenerateInvite(ctx, "tenant-1", "user-admin", []string{"group-1"}, 7)
	after := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("sent"))

	require.NoError(t, err)
	assert.NotEmpty(t, out.ID)
	assert.NotEmpty(t, out.Token)
	assert.True(t, out.ExpiresAt.After(time.Now()))
	assert.Equal(t, []string{"group-1"}, out.GroupIDs)

	// Verify it was persisted
	assert.Len(t, inviteRepo.tokens, 1)

	// Counter must have incremented by exactly 1
	assert.Equal(t, before+1, after)
}

func TestGenerateInvite_DefaultExpiry(t *testing.T) {
	uc, _, _, _ := setupInviteUC()
	ctx := context.Background()

	out, err := uc.GenerateInvite(ctx, "tenant-1", "user-admin", []string{"group-1"}, 0)

	require.NoError(t, err)
	// Should expire approximately 7 days from now
	expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
	diff := out.ExpiresAt.Sub(expectedExpiry)
	assert.Less(t, diff.Abs().Seconds(), float64(5), "expiry should be ~7 days from now")
}

func TestPreviewInvite_Valid(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	invite, err := domain.NewInviteToken("inv-prev", "tenant-7", "creator-1", []string{"group-1"}, time.Now().Add(48*time.Hour))
	require.NoError(t, err)
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	out, err := uc.PreviewInvite(ctx, invite.Token)
	require.NoError(t, err)
	assert.Equal(t, "tenant-7", out.TenantID)
	assert.True(t, out.ExpiresAt.After(time.Now()))
}

func TestPreviewInvite_NotFound(t *testing.T) {
	uc, _, _, _ := setupInviteUC()
	_, err := uc.PreviewInvite(context.Background(), "token-inexistente")
	assert.ErrorIs(t, err, domain.ErrInviteTokenNotFound)
}

func TestPreviewInvite_Expired(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	invite, err := domain.NewInviteToken("inv-exp", "tenant-1", "creator-1", []string{"group-1"}, time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	invite.ExpiresAt = time.Now().Add(-1 * time.Hour) // força expiração
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	_, err = uc.PreviewInvite(context.Background(), invite.Token)
	assert.ErrorIs(t, err, domain.ErrInviteTokenExpired)
}

func TestPreviewInvite_AlreadyUsed(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	invite, err := domain.NewInviteToken("inv-used", "tenant-1", "creator-1", []string{"group-1"}, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	invite.MarkUsed("user-1")
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	_, err = uc.PreviewInvite(context.Background(), invite.Token)
	assert.ErrorIs(t, err, domain.ErrInviteTokenUsed)
}

func TestAcceptInvite_Success_NewUser(t *testing.T) {
	uc, inviteRepo, userRepo, userTenantRepo := setupInviteUC()
	ctx := context.Background()

	// Create a valid invite
	invite, err := domain.NewInviteToken("inv-1", "tenant-1", "creator-1", []string{"group-1"}, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	before := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("accepted"))
	out, err := uc.AcceptInvite(ctx, invite.Token, "Maria", "maria@email.com", "senha123")
	after := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("accepted"))

	require.NoError(t, err)
	assert.NotEmpty(t, out.UserID)
	assert.Equal(t, []string{"group-1"}, out.GroupIDs)

	// User should be created
	_, existsErr := userRepo.FindByEmail(ctx, "maria@email.com")
	assert.NoError(t, existsErr)

	// User should be associated to tenant
	ut, assocErr := userTenantRepo.FindByUserAndTenant(ctx, out.UserID, "tenant-1")
	assert.NoError(t, assocErr)
	assert.NotNil(t, ut)

	// Token should be marked as used
	usedInvite := inviteRepo.tokens[invite.ID]
	assert.True(t, usedInvite.IsUsed())
	assert.Equal(t, out.UserID, usedInvite.UsedBy)

	// Counter must have incremented by exactly 1
	assert.Equal(t, before+1, after)
}

func TestAcceptInvite_Success_ExistingUser(t *testing.T) {
	uc, inviteRepo, userRepo, userTenantRepo := setupInviteUC()
	ctx := context.Background()

	// Pre-existing user
	existingUser := &domain.User{
		ID: "existing-user-1", Name: "João", Email: "joao@email.com",
		PasswordHash: "hashed:old", Role: domain.UserRoleUser, Status: domain.UserStatusActive,
	}
	userRepo.users[existingUser.Email] = existingUser

	// Create a valid invite
	invite, err := domain.NewInviteToken("inv-1", "tenant-1", "creator-1", []string{"group-2"}, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	before := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("accepted"))
	out, err := uc.AcceptInvite(ctx, invite.Token, "João", "joao@email.com", "irrelevant")
	after := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("accepted"))

	require.NoError(t, err)
	assert.Equal(t, "existing-user-1", out.UserID)

	// Should be associated to tenant
	ut, assocErr := userTenantRepo.FindByUserAndTenant(ctx, "existing-user-1", "tenant-1")
	assert.NoError(t, assocErr)
	assert.NotNil(t, ut)

	// Counter must have incremented by exactly 1
	assert.Equal(t, before+1, after)
}

func TestAcceptInvite_ExpiredToken(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	// Create an already-expired invite (expiry in the past)
	invite, err := domain.NewInviteToken("inv-expired", "tenant-1", "creator-1", []string{"group-1"}, time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	before := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("expired"))

	_, err = uc.AcceptInvite(ctx, invite.Token, "Ana", "ana@email.com", "senha")

	assert.ErrorIs(t, err, domain.ErrInviteTokenExpired)

	// Counter for outcome=expired must increment by exactly 1 on the
	// redemption-time observation of the expired token.
	after := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("expired"))
	assert.Equal(t, before+1, after, "invites_total{outcome=expired} should increment once")
}

func TestAcceptInvite_UsedToken(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	// Create an already-used invite
	invite, err := domain.NewInviteToken("inv-used", "tenant-1", "creator-1", []string{"group-1"}, time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	invite.MarkUsed("some-user")
	inviteRepo.tokens[invite.ID] = invite
	inviteRepo.byToken[invite.Token] = invite

	_, err = uc.AcceptInvite(ctx, invite.Token, "Carlos", "carlos@email.com", "senha")

	assert.ErrorIs(t, err, domain.ErrInviteTokenUsed)
}

func TestAcceptInvite_TokenNotFound(t *testing.T) {
	uc, _, _, _ := setupInviteUC()
	ctx := context.Background()

	_, err := uc.AcceptInvite(ctx, "nonexistent-token", "Name", "name@email.com", "senha")

	assert.ErrorIs(t, err, domain.ErrInviteTokenNotFound)
}

func TestListInvites_Success(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	inv1, _ := domain.NewInviteToken("inv-1", "tenant-1", "creator-1", []string{"g1"}, time.Now().Add(24*time.Hour))
	inv2, _ := domain.NewInviteToken("inv-2", "tenant-1", "creator-1", []string{"g2"}, time.Now().Add(48*time.Hour))
	inv3, _ := domain.NewInviteToken("inv-3", "tenant-other", "creator-1", []string{"g3"}, time.Now().Add(24*time.Hour))
	inviteRepo.tokens[inv1.ID] = inv1
	inviteRepo.byToken[inv1.Token] = inv1
	inviteRepo.tokens[inv2.ID] = inv2
	inviteRepo.byToken[inv2.Token] = inv2
	inviteRepo.tokens[inv3.ID] = inv3
	inviteRepo.byToken[inv3.Token] = inv3

	out, err := uc.ListInvites(ctx, "tenant-1")

	require.NoError(t, err)
	assert.Len(t, out, 2)
}

func TestRevokeInvite_Success(t *testing.T) {
	uc, inviteRepo, _, _ := setupInviteUC()
	ctx := context.Background()

	inv, _ := domain.NewInviteToken("inv-1", "tenant-1", "creator-1", []string{"g1"}, time.Now().Add(24*time.Hour))
	inviteRepo.tokens[inv.ID] = inv
	inviteRepo.byToken[inv.Token] = inv

	before := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("revoked"))
	err := uc.RevokeInvite(ctx, "inv-1")
	after := testutil.ToFloat64(infra.InvitesTotal.WithLabelValues("revoked"))

	require.NoError(t, err)
	assert.Empty(t, inviteRepo.tokens)

	// Counter must have incremented by exactly 1
	assert.Equal(t, before+1, after)
}
