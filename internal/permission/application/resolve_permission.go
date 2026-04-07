package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// ResolvePermissionUseCase checks whether a user has a given permission via
// owner bypass, admin bypass, individual permissions, or group memberships.
type ResolvePermissionUseCase struct {
	permissions domain.PermissionRepository
	userGroups  domain.UserGroupRepository
	ownerCheck  domain.OwnerChecker
	adminCheck  domain.AdminChecker
}

func NewResolvePermissionUseCase(
	permissions domain.PermissionRepository,
	userGroups domain.UserGroupRepository,
	ownerCheck domain.OwnerChecker,
	adminCheck domain.AdminChecker,
) *ResolvePermissionUseCase {
	return &ResolvePermissionUseCase{
		permissions: permissions,
		userGroups:  userGroups,
		ownerCheck:  ownerCheck,
		adminCheck:  adminCheck,
	}
}

// HasPermission resolves whether userID holds resource+action in tenantID.
// Resolution order:
//  1. Owner bypass
//  2. Admin bypass
//  3. Individual (user-level) permissions
//  4. Group permissions (union across all groups the user belongs to)
func (uc *ResolvePermissionUseCase) HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error) {
	// 1. Owner bypass
	isOwner, err := uc.ownerCheck.IsOwner(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	if isOwner {
		return true, nil
	}

	// 2. Admin bypass
	isAdmin, err := uc.adminCheck.IsAdmin(ctx, userID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}

	// 3. Individual permissions
	individualPerms, err := uc.permissions.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return false, err
	}
	for _, p := range individualPerms {
		if p.Resource == resource && p.Action == action {
			return true, nil
		}
	}

	// 4. Group permissions
	userGroupList, err := uc.userGroups.FindByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	if len(userGroupList) == 0 {
		return false, nil
	}

	groupIDs := make([]string, len(userGroupList))
	for i, ug := range userGroupList {
		groupIDs[i] = ug.GroupID
	}

	groupPerms, err := uc.permissions.FindByGroupIDs(ctx, groupIDs)
	if err != nil {
		return false, err
	}
	for _, p := range groupPerms {
		if p.Resource == resource && p.Action == action {
			return true, nil
		}
	}

	return false, nil
}
