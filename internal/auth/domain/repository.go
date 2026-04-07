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
	FindByTenantID(ctx context.Context, tenantID string) ([]*UserTenant, error)
	FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*UserTenant, error)
	UpdateIsOwner(ctx context.Context, userID, tenantID string, isOwner bool) error
	UpdateWhatsAppID(ctx context.Context, userID, tenantID string, whatsAppID string) error
	RemoveFromTenant(ctx context.Context, userID, tenantID string) error
	IsOwner(ctx context.Context, userID, tenantID string) (bool, error)
}

type InviteTokenRepository interface {
	Create(ctx context.Context, token *InviteToken) error
	FindByToken(ctx context.Context, token string) (*InviteToken, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]*InviteToken, error)
	Update(ctx context.Context, token *InviteToken) error
	Delete(ctx context.Context, id string) error
}

type LoadBalanceConfigRepository interface {
	CreateOrUpdate(ctx context.Context, cfg *LoadBalanceConfig) error
	FindByGroupID(ctx context.Context, tenantID, groupID string) (*LoadBalanceConfig, error)
	FindByTenantID(ctx context.Context, tenantID string) ([]*LoadBalanceConfig, error)
	Update(ctx context.Context, cfg *LoadBalanceConfig) error
}
