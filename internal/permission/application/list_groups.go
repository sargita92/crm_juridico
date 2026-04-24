package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// ListGroupsUseCase retrieves all permission groups for a tenant.
type ListGroupsUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewListGroupsUseCase(groups domain.PermissionGroupRepository) *ListGroupsUseCase {
	return &ListGroupsUseCase{groups: groups}
}

func (uc *ListGroupsUseCase) Execute(ctx context.Context, tenantID string) ([]GroupOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.list_groups",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

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
