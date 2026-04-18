package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

// GroupTenantCheckerAdapter checks whether a permission group belongs to a tenant
// by querying the permission_groups table directly. This avoids a cross-module
// import of the permission domain from auth.
type GroupTenantCheckerAdapter struct {
	db *gorm.DB
}

func NewGroupTenantCheckerAdapter(db *gorm.DB) *GroupTenantCheckerAdapter {
	return &GroupTenantCheckerAdapter{db: db}
}

func (a *GroupTenantCheckerAdapter) BelongsToTenant(ctx context.Context, tenantID, groupID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Table("permission_groups").
		Where("id = ? AND tenant_id = ?", groupID, tenantID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
