package infrastructure

import (
	"context"
	"errors"
	"sort"

	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// LoadBalancePicker implements funnel/domain.ResponsiblePicker.
type LoadBalancePicker struct {
	groupFunnelRepo permdomain.GroupFunnelRepository
	lbRepo          authdomain.LoadBalanceConfigRepository
	userGroupRepo   permdomain.UserGroupRepository
	userTenantRepo  authdomain.UserTenantRepository
	loadCounter     funneldomain.LeadLoadCounter
	log             *zap.Logger
}

// NewLoadBalancePicker wires all dependencies needed to decide which user
// should receive a newly created lead. A nil logger is replaced with a no-op.
func NewLoadBalancePicker(
	groupFunnelRepo permdomain.GroupFunnelRepository,
	lbRepo authdomain.LoadBalanceConfigRepository,
	userGroupRepo permdomain.UserGroupRepository,
	userTenantRepo authdomain.UserTenantRepository,
	loadCounter funneldomain.LeadLoadCounter,
	log *zap.Logger,
) *LoadBalancePicker {
	if log == nil {
		log = zap.NewNop()
	}
	return &LoadBalancePicker{
		groupFunnelRepo: groupFunnelRepo,
		lbRepo:          lbRepo,
		userGroupRepo:   userGroupRepo,
		userTenantRepo:  userTenantRepo,
		loadCounter:     loadCounter,
		log:             log,
	}
}

// PickForFunnel implements ResponsiblePicker.
func (p *LoadBalancePicker) PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
	// 1. Find all groups associated with this funnel that cover the column.
	groups, err := p.groupFunnelRepo.FindByFunnelID(ctx, funnelID)
	if err != nil {
		return p.fallbackToOwner(ctx, tenantID, "group_lookup_error")
	}
	covering := make([]permdomain.GroupFunnel, 0, len(groups))
	for _, gf := range groups {
		if gf.CoversColumn(columnID) {
			covering = append(covering, gf)
		}
	}
	if len(covering) == 0 {
		return p.fallbackToOwner(ctx, tenantID, "no_group")
	}

	// 2. Filter to groups whose LoadBalanceConfig is ACTIVE.
	type active struct {
		groupID string
		cfg     *authdomain.LoadBalanceConfig
	}
	var actives []active
	for _, gf := range covering {
		cfg, err := p.lbRepo.FindByGroupID(ctx, tenantID, gf.GroupID)
		if err != nil {
			if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
				continue
			}
			return p.fallbackToOwner(ctx, tenantID, "lb_lookup_error")
		}
		if cfg != nil && cfg.Active {
			actives = append(actives, active{gf.GroupID, cfg})
		}
	}
	if len(actives) == 0 {
		return p.fallbackToOwner(ctx, tenantID, "no_active_config")
	}
	if len(actives) > 1 {
		p.log.Error("load_balance.pick multiple_active_groups — check uniqueness rule",
			zap.String("tenant_id", tenantID), zap.String("funnel_id", funnelID), zap.String("column_id", columnID),
		)
		return p.fallbackToOwner(ctx, tenantID, "multiple_active_groups")
	}

	chosen := actives[0]

	// 3. Fetch group members and filter to active tenant members.
	ugs, err := p.userGroupRepo.FindByGroupID(ctx, chosen.groupID)
	if err != nil {
		return p.fallbackToOwner(ctx, tenantID, "member_lookup_error")
	}
	members := make([]string, 0, len(ugs))
	for _, ug := range ugs {
		if _, err := p.userTenantRepo.FindByUserAndTenant(ctx, ug.UserID, tenantID); err == nil {
			members = append(members, ug.UserID)
		}
	}
	if len(members) == 0 {
		return p.fallbackToOwner(ctx, tenantID, "no_active_members")
	}

	// 4. Apply the algorithm.
	pickedUserID, err := p.applyAlgorithm(ctx, tenantID, chosen.cfg, members)
	if err != nil {
		return p.fallbackToOwner(ctx, tenantID, "algorithm_error")
	}
	algorithm := string(chosen.cfg.Algorithm)

	return funneldomain.PickResult{
		UserID:    pickedUserID,
		Algorithm: algorithm,
		GroupID:   chosen.groupID,
		Outcome:   funneldomain.PickOutcomePicked,
	}, nil
}

func (p *LoadBalancePicker) applyAlgorithm(ctx context.Context, tenantID string, cfg *authdomain.LoadBalanceConfig, members []string) (string, error) {
	sort.Strings(members) // deterministic order for round_robin and tiebreaks

	switch cfg.Algorithm {
	case authdomain.AlgorithmRoundRobin:
		idx := cfg.LastIndex % len(members)
		picked := members[idx]
		cfg.IncrementIndex()
		if err := p.lbRepo.Update(ctx, cfg); err != nil {
			return "", err
		}
		return picked, nil
	case authdomain.AlgorithmLeastLoad:
		return members[0], nil // placeholder — Task 8
	case authdomain.AlgorithmRandom:
		return members[0], nil // placeholder — Task 9
	default:
		return members[0], nil
	}
}

func (p *LoadBalancePicker) fallbackToOwner(ctx context.Context, tenantID, reason string) (funneldomain.PickResult, error) {
	uts, err := p.userTenantRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return funneldomain.PickResult{}, err
	}
	for _, ut := range uts {
		if ut.IsOwner {
			p.log.Info("load_balance.pick fallback_owner",
				zap.String("tenant_id", tenantID),
				zap.String("reason", reason),
				zap.String("picked_user_id", ut.UserID),
			)
			return funneldomain.PickResult{
				UserID:  ut.UserID,
				Outcome: funneldomain.PickOutcomeFallbackOwner,
			}, nil
		}
	}
	return funneldomain.PickResult{}, funneldomain.ErrNoResponsibleAvailable
}

// sanity: ensure interface compliance at compile time
var _ funneldomain.ResponsiblePicker = (*LoadBalancePicker)(nil)
