package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type ListUserTenantsUseCase struct {
	userTenantRepo domain.UserTenantRepository
	tenantRepo     tenantdomain.TenantRepository
}

func NewListUserTenantsUseCase(
	userTenantRepo domain.UserTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
) *ListUserTenantsUseCase {
	return &ListUserTenantsUseCase{
		userTenantRepo: userTenantRepo,
		tenantRepo:     tenantRepo,
	}
}

func (uc *ListUserTenantsUseCase) Execute(ctx context.Context, userID string, role domain.UserRole) ([]tenantdomain.Tenant, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.list_user_tenants",
		attribute.String("user.id", userID),
	)
	defer span.End()

	if role == domain.UserRoleAdmin {
		return uc.tenantRepo.FindAll(ctx)
	}

	ids, err := uc.userTenantRepo.FindTenantIDsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return uc.tenantRepo.FindByIDs(ctx, ids)
}
