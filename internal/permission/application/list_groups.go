package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// ListGroupsUseCase retrieves all permission groups for a tenant.
type ListGroupsUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewListGroupsUseCase(groups domain.PermissionGroupRepository) *ListGroupsUseCase {
	return &ListGroupsUseCase{groups: groups}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context, tenantID string) ([]GroupOutput, error) {
	list, err := uc.groups.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupOutput, len(list))
	for i, g := range list {
		out[i] = GroupOutput{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
		}
	}
	return out, nil
}
