package infrastructure

import (
	"context"

	"gorm.io/gorm"
)

// OwnerCheckerAdapter checks whether a user is the owner of a tenant.
// Plan 2 will add the is_owner column; until then this always returns false.
type OwnerCheckerAdapter struct {
	db *gorm.DB
}

// NewOwnerCheckerAdapter creates an OwnerCheckerAdapter.
func NewOwnerCheckerAdapter(db *gorm.DB) *OwnerCheckerAdapter {
	return &OwnerCheckerAdapter{db: db}
}

// IsOwner returns true when the user_tenants row has is_owner = true.
func (a *OwnerCheckerAdapter) IsOwner(ctx context.Context, userID, tenantID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).Table("user_tenants").
		Where("user_id = ? AND tenant_id = ? AND is_owner = ?", userID, tenantID, true).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AdminCheckerAdapter checks whether a user holds the "admin" role.
type AdminCheckerAdapter struct {
	db *gorm.DB
}

// NewAdminCheckerAdapter creates an AdminCheckerAdapter.
func NewAdminCheckerAdapter(db *gorm.DB) *AdminCheckerAdapter {
	return &AdminCheckerAdapter{db: db}
}

// IsAdmin returns true when the user's role is "admin" in the users table.
func (a *AdminCheckerAdapter) IsAdmin(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := a.db.WithContext(ctx).
		Table("users").
		Where("id = ? AND role = ?", userID, "admin").
		Count(&count).Error
	return count > 0, err
}
