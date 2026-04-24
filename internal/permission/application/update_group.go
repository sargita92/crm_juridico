package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// UpdateGroupInput holds the data required to update an existing group.
type UpdateGroupInput struct {
	TenantID    string
	ID          string
	Name        string
	Description string
}

// UpdateGroupUseCase updates the name and description of a PermissionGroup.
type UpdateGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewUpdateGroupUseCase(groups domain.PermissionGroupRepository) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{groups: groups}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, input UpdateGroupInput) (*GroupOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.update_group",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("group.id", input.ID),
	)
	defer span.End()

	group, err := uc.groups.FindByIDAndTenantID(ctx, input.ID, input.TenantID)
	if err != nil {
		return nil, err
	}
	if err := group.Update(input.Name, input.Description); err != nil {
		return nil, err
	}
	if err := uc.groups.Update(ctx, group); err != nil {
		return nil, err
	}
	return &GroupOutput{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
	}, nil
}
