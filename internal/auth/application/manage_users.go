package application

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

var ErrCannotRemoveOwner = errors.New("cannot remove the tenant owner")

// UserOutput carries tenant-scoped user data.
type UserOutput struct {
	ID         string
	Name       string
	Email      string
	IsOwner    bool
	WhatsAppID string
}

// ManageUsersUseCase handles user management within a tenant.
type ManageUsersUseCase struct {
	userRepo       domain.UserRepository
	userTenantRepo domain.UserTenantRepository
}

// NewManageUsersUseCase creates a new ManageUsersUseCase.
func NewManageUsersUseCase(
	userRepo domain.UserRepository,
	userTenantRepo domain.UserTenantRepository,
) *ManageUsersUseCase {
	return &ManageUsersUseCase{
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
	}
}

// ListTenantUsers returns all users belonging to the given tenant.
func (uc *ManageUsersUseCase) ListTenantUsers(ctx context.Context, tenantID string) ([]UserOutput, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.list_tenant_users",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	userTenants, err := uc.userTenantRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	out := make([]UserOutput, 0, len(userTenants))
	for _, ut := range userTenants {
		user, err := uc.userRepo.FindByID(ctx, ut.UserID)
		if err != nil {
			return nil, err
		}

		out = append(out, UserOutput{
			ID:         user.ID,
			Name:       user.Name,
			Email:      user.Email,
			IsOwner:    ut.IsOwner,
			WhatsAppID: ut.WhatsAppID,
		})
	}

	return out, nil
}

// RemoveFromTenant removes a user from the tenant. Returns an error if the user is the owner.
func (uc *ManageUsersUseCase) RemoveFromTenant(ctx context.Context, userID, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.remove_from_tenant",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	isOwner, err := uc.userTenantRepo.IsOwner(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if isOwner {
		return ErrCannotRemoveOwner
	}

	return uc.userTenantRepo.RemoveFromTenant(ctx, userID, tenantID)
}

// SetWhatsAppID updates the WhatsApp number linked to a user within a tenant.
func (uc *ManageUsersUseCase) SetWhatsAppID(ctx context.Context, userID, tenantID, whatsAppID string) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.set_whatsapp_id",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	return uc.userTenantRepo.UpdateWhatsAppID(ctx, userID, tenantID, whatsAppID)
}
