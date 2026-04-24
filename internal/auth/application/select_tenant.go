package application

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

var ErrTenantNotAllowed = errors.New("user does not have access to this tenant")

type SelectTenantInput struct {
	UserID   string
	TenantID string
	Role     domain.UserRole
}

type SelectTenantOutput struct {
	Token string
}

type SelectTenantUseCase struct {
	userTenantRepo domain.UserTenantRepository
	tenantRepo     tenantdomain.TenantRepository
	tokenProvider  domain.TokenProvider
}

func NewSelectTenantUseCase(
	userTenantRepo domain.UserTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
	tokenProvider domain.TokenProvider,
) *SelectTenantUseCase {
	return &SelectTenantUseCase{
		userTenantRepo: userTenantRepo,
		tenantRepo:     tenantRepo,
		tokenProvider:  tokenProvider,
	}
}

func (uc *SelectTenantUseCase) Execute(ctx context.Context, input SelectTenantInput) (*SelectTenantOutput, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.select_tenant",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("user.id", input.UserID),
	)
	defer span.End()

	tenant, err := uc.tenantRepo.FindByID(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}

	if !tenant.IsActive() {
		if tenant.IsBlocked() {
			return nil, tenantdomain.ErrTenantBlocked
		}
		return nil, tenantdomain.ErrTenantInactive
	}

	// Admin can access any tenant
	if input.Role != domain.UserRoleAdmin {
		tenantIDs, err := uc.userTenantRepo.FindTenantIDsByUserID(ctx, input.UserID)
		if err != nil {
			return nil, err
		}

		allowed := false
		for _, id := range tenantIDs {
			if id == input.TenantID {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrTenantNotAllowed
		}
	}

	token, err := uc.tokenProvider.Generate(domain.TokenClaims{
		UserID:   input.UserID,
		Role:     input.Role,
		TenantID: input.TenantID,
	})
	if err != nil {
		return nil, err
	}

	return &SelectTenantOutput{Token: token}, nil
}
