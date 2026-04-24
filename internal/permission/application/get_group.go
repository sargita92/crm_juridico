package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// GetGroupUseCase retrieves a single permission group by ID, scoped to a tenant.
type GetGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewGetGroupUseCase(groups domain.PermissionGroupRepository) *GetGroupUseCase {
	return &GetGroupUseCase{groups: groups}
}

func (uc *GetGroupUseCase) Execute(ctx context.Context, tenantID, id string) (*GroupOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.get_group",
		attribute.String("tenant.id", tenantID),
		attribute.String("group.id", id),
	)
	defer span.End()

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
