package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	Token    string
	UserID   string
	UserName string
	Role     domain.UserRole
	Tenants  []tenantdomain.Tenant
}

type LoginUseCase struct {
	userRepo       domain.UserRepository
	userTenantRepo domain.UserTenantRepository
	tenantRepo     tenantdomain.TenantRepository
	hasher         domain.PasswordHasher
	tokenProvider  domain.TokenProvider
}

func NewLoginUseCase(
	userRepo domain.UserRepository,
	userTenantRepo domain.UserTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
	hasher domain.PasswordHasher,
	tokenProvider domain.TokenProvider,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		tenantRepo:     tenantRepo,
		hasher:         hasher,
		tokenProvider:  tokenProvider,
	}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.login",
		attribute.String("auth.email", input.Email),
	)
	defer span.End()

	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.IsActive() {
		return nil, domain.ErrInvalidCredentials
	}

	if err := uc.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	var tenants []tenantdomain.Tenant
	if user.IsAdmin() {
		// Admin always sees all active tenants
		tenants, err = uc.tenantRepo.FindAll(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		tenantIDs, err := uc.userTenantRepo.FindTenantIDsByUserID(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if len(tenantIDs) > 0 {
			tenants, err = uc.tenantRepo.FindByIDs(ctx, tenantIDs)
			if err != nil {
				return nil, err
			}
		}
	}

	claims := domain.TokenClaims{
		UserID: user.ID,
		Role:   user.Role,
	}

	// Non-admin with exactly 1 tenant: include tenant_id in token directly
	if !user.IsAdmin() && len(tenants) == 1 && tenants[0].IsActive() {
		claims.TenantID = tenants[0].ID
	}

	token, err := uc.tokenProvider.Generate(claims)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token:    token,
		UserID:   user.ID,
		UserName: user.Name,
		Role:     user.Role,
		Tenants:  tenants,
	}, nil
}
