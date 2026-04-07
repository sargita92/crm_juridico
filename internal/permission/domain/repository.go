package domain

import "context"

type PermissionGroupRepository interface {
	Create(ctx context.Context, group *PermissionGroup) error
	FindByID(ctx context.Context, id string) (*PermissionGroup, error)
	FindByIDAndTenantID(ctx context.Context, id, tenantID string) (*PermissionGroup, error)
	Update(ctx context.Context, group *PermissionGroup) error
	Delete(ctx context.Context, id string) error
	FindByTenantID(ctx context.Context, tenantID string) ([]PermissionGroup, error)
}

type UserGroupRepository interface {
	Create(ctx context.Context, ug *UserGroup) error
	Delete(ctx context.Context, userID, groupID string) error
	FindByGroupID(ctx context.Context, groupID string) ([]UserGroup, error)
	FindByUserAndTenant(ctx context.Context, userID, tenantID string) ([]UserGroup, error)
	Exists(ctx context.Context, userID, groupID string) (bool, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, p *Permission) error
	Delete(ctx context.Context, id string) error
	FindByGroupID(ctx context.Context, groupID string) ([]Permission, error)
	FindByUserID(ctx context.Context, tenantID, userID string) ([]Permission, error)
	FindByGroupIDs(ctx context.Context, groupIDs []string) ([]Permission, error)
	DeleteByGroupAndResource(ctx context.Context, groupID, resource, action string) error
	DeleteByUserAndResource(ctx context.Context, tenantID, userID, resource, action string) error
}

type ViewProfileRepository interface {
	CreateOrUpdate(ctx context.Context, vp *ViewProfile) error
	FindByGroupID(ctx context.Context, groupID string) ([]ViewProfile, error)
	FindByGroupAndFunnel(ctx context.Context, groupID, funnelID string) (*ViewProfile, error)
	FindByGroupIDs(ctx context.Context, groupIDs []string, funnelID string) ([]ViewProfile, error)
	Delete(ctx context.Context, groupID, funnelID string) error
}

type GroupFunnelRepository interface {
	CreateOrUpdate(ctx context.Context, gf *GroupFunnel) error
	FindByGroupID(ctx context.Context, groupID string) ([]GroupFunnel, error)
	FindByFunnelID(ctx context.Context, funnelID string) ([]GroupFunnel, error)
	FindByFunnelAndColumn(ctx context.Context, funnelID, columnID string) ([]GroupFunnel, error)
	Delete(ctx context.Context, groupID, funnelID string) error
}

// OwnerChecker determines whether a user is the owner of a tenant.
type OwnerChecker interface {
	IsOwner(ctx context.Context, userID, tenantID string) (bool, error)
}

// AdminChecker determines whether a user holds global admin privileges.
type AdminChecker interface {
	IsAdmin(ctx context.Context, userID string) (bool, error)
}
