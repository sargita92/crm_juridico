package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// UpdateGroupInput holds the data required to update an existing group.
type UpdateGroupInput struct {
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
	group, err := uc.groups.FindByID(ctx, input.ID)
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
