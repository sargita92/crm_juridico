package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
)

var ErrUserAlreadyInTenant = errors.New("user is already associated to this tenant")

// InviteOutput carries the data returned after generating an invite.
type InviteOutput struct {
	ID        string
	Token     string
	ExpiresAt time.Time
	GroupIDs  []string
}

// AcceptOutput carries the data returned after accepting an invite.
type AcceptOutput struct {
	UserID   string
	GroupIDs []string
}

// InviteUserUseCase handles invite lifecycle operations.
type InviteUserUseCase struct {
	inviteRepo     domain.InviteTokenRepository
	userRepo       domain.UserRepository
	userTenantRepo domain.UserTenantRepository
	hasher         domain.PasswordHasher
}

// NewInviteUserUseCase creates a new InviteUserUseCase.
func NewInviteUserUseCase(
	inviteRepo domain.InviteTokenRepository,
	userRepo domain.UserRepository,
	userTenantRepo domain.UserTenantRepository,
	hasher domain.PasswordHasher,
) *InviteUserUseCase {
	return &InviteUserUseCase{
		inviteRepo:     inviteRepo,
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		hasher:         hasher,
	}
}

// GenerateInvite creates and persists a new invite token for the given tenant.
// If expiresInDays is 0 the default of 7 days is used.
func (uc *InviteUserUseCase) GenerateInvite(
	ctx context.Context,
	tenantID, createdBy string,
	groupIDs []string,
	expiresInDays int,
) (*InviteOutput, error) {
	if expiresInDays <= 0 {
		expiresInDays = 7
	}

	expiresAt := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)

	invite, err := domain.NewInviteToken(uuid.NewString(), tenantID, createdBy, groupIDs, expiresAt)
	if err != nil {
		return nil, err
	}

	if err := uc.inviteRepo.Create(ctx, invite); err != nil {
		return nil, err
	}

	infrastructure.InvitesTotal.WithLabelValues("sent").Inc()
	return &InviteOutput{
		ID:        invite.ID,
		Token:     invite.Token,
		ExpiresAt: invite.ExpiresAt,
		GroupIDs:  invite.GroupIDs,
	}, nil
}

// AcceptInvite redeems an invite token, creating or associating the user to the tenant.
// GroupIDs assignment to permission groups will be handled via adapter later.
func (uc *InviteUserUseCase) AcceptInvite(
	ctx context.Context,
	token, name, email, password string,
) (*AcceptOutput, error) {
	invite, err := uc.inviteRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if err := invite.Validate(); err != nil {
		return nil, err
	}

	exists, err := uc.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var userID string

	if !exists {
		// New user: hash password and create
		hash, err := uc.hasher.Hash(password)
		if err != nil {
			return nil, err
		}

		user, err := domain.NewUser(uuid.NewString(), name, email, hash, domain.UserRoleUser)
		if err != nil {
			return nil, err
		}

		if err := uc.userRepo.Create(ctx, user); err != nil {
			return nil, err
		}

		userID = user.ID
	} else {
		// Existing user: find and reuse
		user, err := uc.userRepo.FindByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		userID = user.ID
	}

	// Associate user to tenant if not already associated
	existing, err := uc.userTenantRepo.FindByUserAndTenant(ctx, userID, invite.TenantID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, err
	}

	if existing == nil {
		if err := uc.userTenantRepo.Associate(ctx, userID, invite.TenantID); err != nil {
			return nil, err
		}
	}

	// Mark invite as used
	invite.MarkUsed(userID)
	if err := uc.inviteRepo.Update(ctx, invite); err != nil {
		return nil, err
	}

	infrastructure.InvitesTotal.WithLabelValues("accepted").Inc()
	return &AcceptOutput{
		UserID:   userID,
		GroupIDs: invite.GroupIDs,
	}, nil
}

// ListInvites returns all invite tokens for the given tenant.
func (uc *InviteUserUseCase) ListInvites(ctx context.Context, tenantID string) ([]InviteOutput, error) {
	invites, err := uc.inviteRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	out := make([]InviteOutput, 0, len(invites))
	for _, inv := range invites {
		out = append(out, InviteOutput{
			ID:        inv.ID,
			Token:     inv.Token,
			ExpiresAt: inv.ExpiresAt,
			GroupIDs:  inv.GroupIDs,
		})
	}
	return out, nil
}

// RevokeInvite deletes an invite token by ID.
func (uc *InviteUserUseCase) RevokeInvite(ctx context.Context, id string) error {
	if err := uc.inviteRepo.Delete(ctx, id); err != nil {
		return err
	}
	infrastructure.InvitesTotal.WithLabelValues("revoked").Inc()
	return nil
}
