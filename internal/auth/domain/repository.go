package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type UserTenantRepository interface {
	Associate(ctx context.Context, userID, tenantID string) error
	FindTenantIDsByUserID(ctx context.Context, userID string) ([]string, error)
}
