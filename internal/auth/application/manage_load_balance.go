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
	repo           domain.LoadBalanceConfigRepository
	groupChecker   GroupTenantChecker
	overlapChecker GroupColumnOverlapChecker
}

func NewManageLoadBalanceUseCase(
	repo domain.LoadBalanceConfigRepository,
	groupChecker GroupTenantChecker,
	overlapChecker GroupColumnOverlapChecker,
) *ManageLoadBalanceUseCase {
	return &ManageLoadBalanceUseCase{
		repo:           repo,
		groupChecker:   groupChecker,
		overlapChecker: overlapChecker,
	}
}

// SetOverlapChecker allows late injection of the overlap checker after
// construction. This is used by the wiring layer when the checker depends on
// repositories owned by a different module (chicken-and-egg resolution).
func (uc *ManageLoadBalanceUseCase) SetOverlapChecker(checker GroupColumnOverlapChecker) {
	uc.overlapChecker = checker
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

	if in.Active && uc.overlapChecker != nil {
		overlap, _, err := uc.overlapChecker.HasActiveOverlap(ctx, in.TenantID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if overlap {
			// Overlapping group IDs are internal data; do not leak them in the
			// user-facing error. The wiring layer can log them at its boundary
			// if operational visibility is needed.
			return nil, ErrActiveLoadBalanceOverlap
		}
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
