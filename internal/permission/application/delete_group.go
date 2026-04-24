package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// DeleteGroupUseCase removes a PermissionGroup by ID, scoped to a tenant.
type DeleteGroupUseCase struct {
	groups domain.PermissionGroupRepository
}

func NewDeleteGroupUseCase(groups domain.PermissionGroupRepository) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{groups: groups}
}

func (uc *DeleteGroupUseCase) Execute(ctx context.Context, tenantID, id string) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.delete_group",
		attribute.String("tenant.id", tenantID),
		attribute.String("group.id", id),
	)
	defer span.End()

	// Verify the group belongs to the tenant before deleting (prevent leaking existence).
	if _, err := uc.groups.FindByIDAndTenantID(ctx, id, tenantID); err != nil {
		return err
	}
	return uc.groups.Delete(ctx, id)
}
