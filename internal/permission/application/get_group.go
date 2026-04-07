package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// GetGroupUseCase retrieves a single permission group by ID, scoped to a tenant.
type GetGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewGetGroupUseCase(groups domain.PermissionGroupRepository) *GetGroupUseCase {
	return &GetGroupUseCase{groups: groups}
}

func (uc *GetGroupUseCase) Execute(ctx context.Context, tenantID, id string) (*GroupOutput, error) {
	group, err := uc.groups.FindByIDAndTenantID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	return &GroupOutput{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
	}, nil
}
