package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// CreateGroupInput holds the data required to create a new permission group.
type CreateGroupInput struct {
	TenantID    string
	Name        string
	Description string
}

// GroupOutput is the read model returned by group use cases.
type GroupOutput struct {
	ID          string
	Name        string
	Description string
}

// CreateGroupUseCase creates a new PermissionGroup within a tenant.
type CreateGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewCreateGroupUseCase(groups domain.PermissionGroupRepository) *CreateGroupUseCase {
	return &CreateGroupUseCase{groups: groups}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*GroupOutput, error) {
	group, err := domain.NewPermissionGroup(uuid.New().String(), input.TenantID, input.Name, input.Description)
	if err != nil {
		return nil, err
	}
	if err := uc.groups.Create(ctx, group); err != nil {
		return nil, err
	}
	return &GroupOutput{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
	}, nil
}
