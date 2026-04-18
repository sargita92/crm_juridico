package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// GroupTenantChecker validates that a group belongs to the given tenant.
// This avoids a hard import of the permission domain from auth.
type GroupTenantChecker interface {
	BelongsToTenant(ctx context.Context, tenantID, groupID string) (bool, error)
}

var ErrGroupNotInTenant = errors.New("group does not belong to tenant")

type SetLoadBalanceInput struct {
	TenantID  string
	GroupID   string
	Algorithm domain.LoadBalanceAlgorithm
	Active    bool
}

type ManageLoadBalanceUseCase struct {
	repo         domain.LoadBalanceConfigRepository
	groupChecker GroupTenantChecker
}

func NewManageLoadBalanceUseCase(
	repo domain.LoadBalanceConfigRepository,
	groupChecker GroupTenantChecker,
) *ManageLoadBalanceUseCase {
	return &ManageLoadBalanceUseCase{repo: repo, groupChecker: groupChecker}
}

func (uc *ManageLoadBalanceUseCase) GetByGroup(ctx context.Context, tenantID, groupID string) (*domain.LoadBalanceConfig, error) {
	ok, err := uc.groupChecker.BelongsToTenant(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGroupNotInTenant
	}
	return uc.repo.FindByGroupID(ctx, tenantID, groupID)
}

func (uc *ManageLoadBalanceUseCase) SetByGroup(ctx context.Context, in SetLoadBalanceInput) (*domain.LoadBalanceConfig, error) {
	ok, err := uc.groupChecker.BelongsToTenant(ctx, in.TenantID, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrGroupNotInTenant
	}

	if err := domain.ValidateAlgorithm(in.Algorithm); err != nil {
		return nil, err
	}

	existing, err := uc.repo.FindByGroupID(ctx, in.TenantID, in.GroupID)
	if err != nil && !errors.Is(err, domain.ErrLoadBalanceNotFound) {
		return nil, err
	}

	var cfg *domain.LoadBalanceConfig
	if existing == nil {
		cfg, err = domain.NewLoadBalanceConfig(uuid.NewString(), in.TenantID, in.GroupID, in.Algorithm)
		if err != nil {
			return nil, err
		}
	} else {
		cfg = existing
		cfg.Algorithm = in.Algorithm
	}
	cfg.Active = in.Active

	if err := uc.repo.CreateOrUpdate(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
