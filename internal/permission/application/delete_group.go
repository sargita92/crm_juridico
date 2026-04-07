package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// DeleteGroupUseCase removes a PermissionGroup by ID, scoped to a tenant.
type DeleteGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewDeleteGroupUseCase(groups domain.PermissionGroupRepository) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{groups: groups}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, tenantID, id string) error {
	// Verify the group belongs to the tenant before deleting (prevent leaking existence).
	if _, err := uc.groups.FindByIDAndTenantID(ctx, id, tenantID); err != nil {
		return err
	}
	return uc.groups.Delete(ctx, id)
}
