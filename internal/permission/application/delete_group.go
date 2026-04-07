package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// DeleteGroupUseCase removes a PermissionGroup by ID.
type DeleteGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewDeleteGroupUseCase(groups domain.PermissionGroupRepository) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{groups: groups}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, id string) error {
	return uc.groups.Delete(ctx, id)
}
