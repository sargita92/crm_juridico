package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
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
