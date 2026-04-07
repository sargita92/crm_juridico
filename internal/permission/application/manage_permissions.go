package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// PermissionInput describes a resource+action pair to be assigned.
type PermissionInput struct {
	Resource string
	Action   string
}

// PermissionOutput is the read model returned after permission queries.
type PermissionOutput struct {
	ID       string
	Resource string
	Action   string
}

// ManagePermissionsUseCase handles group and user permission assignment.
type ManagePermissionsUseCase struct {
	permissions domain.PermissionRepository
}

func NewManagePermissionsUseCase(permissions domain.PermissionRepository) *ManagePermissionsUseCase {
	return &ManagePermissionsUseCase{permissions: permissions}
}

// SetGroupPermissions replaces all permissions for the given group inside tenantID.
// It validates every input first, deletes existing permissions per resource, then
// creates the new set.
func (uc *ManagePermissionsUseCase) SetGroupPermissions(ctx context.Context, tenantID, groupID string, inputs []PermissionInput) error {
	// Validate all inputs before making any changes.
	for _, inp := range inputs {
		if _, err := domain.NewPermission(uuid.New().String(), tenantID, groupID, "", inp.Resource, inp.Action); err != nil {
			return err
		}
	}
	// Track which resources we've already cleared to avoid duplicate deletions.
	cleared := make(map[string]bool)
	for _, inp := range inputs {
		key := inp.Resource + "|" + inp.Action
		if !cleared[key] {
			if err := uc.permissions.DeleteByGroupAndResource(ctx, groupID, inp.Resource, inp.Action); err != nil {
				return err
			}
			cleared[key] = true
		}
	}
	for _, inp := range inputs {
		p, _ := domain.NewPermission(uuid.New().String(), tenantID, groupID, "", inp.Resource, inp.Action)
		if err := uc.permissions.Create(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// SetUserPermissions replaces all permissions for the given user inside tenantID.
func (uc *ManagePermissionsUseCase) SetUserPermissions(ctx context.Context, tenantID, userID string, inputs []PermissionInput) error {
	// Validate all inputs before making any changes.
	for _, inp := range inputs {
		if _, err := domain.NewPermission(uuid.New().String(), tenantID, "", userID, inp.Resource, inp.Action); err != nil {
			return err
		}
	}
	cleared := make(map[string]bool)
	for _, inp := range inputs {
		key := inp.Resource + "|" + inp.Action
		if !cleared[key] {
			if err := uc.permissions.DeleteByUserAndResource(ctx, tenantID, userID, inp.Resource, inp.Action); err != nil {
				return err
			}
			cleared[key] = true
		}
	}
	for _, inp := range inputs {
		p, _ := domain.NewPermission(uuid.New().String(), tenantID, "", userID, inp.Resource, inp.Action)
		if err := uc.permissions.Create(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// GetGroupPermissions returns all permissions currently assigned to a group.
func (uc *ManagePermissionsUseCase) GetGroupPermissions(ctx context.Context, groupID string) ([]PermissionOutput, error) {
	list, err := uc.permissions.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return toPermissionOutputs(list), nil
}

// GetUserPermissions returns all individual permissions assigned to a user
// within a tenant.
func (uc *ManagePermissionsUseCase) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]PermissionOutput, error) {
	list, err := uc.permissions.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return toPermissionOutputs(list), nil
}

func toPermissionOutputs(perms []domain.Permission) []PermissionOutput {
	out := make([]PermissionOutput, len(perms))
	for i, p := range perms {
		out[i] = PermissionOutput{
			ID:       p.ID,
			Resource: p.Resource,
			Action:   p.Action,
		}
	}
	return out
}
