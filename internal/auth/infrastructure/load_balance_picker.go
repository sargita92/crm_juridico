package infrastructure

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"sort"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

// PickForFunnel implements ResponsiblePicker. It wraps pickInternal with
// tracing + metrics so observability stays centralized.
func (p *LoadBalancePicker) PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
	start := time.Now()
	tracer := otel.Tracer("crm.load_balance")
	ctx, span := tracer.Start(ctx, "load_balance.pick")
	defer span.End()
	span.SetAttributes(
		attribute.String("tenant_id", tenantID),
		attribute.String("funnel_id", funnelID),
		attribute.String("column_id", columnID),
	)

	res, err := p.pickInternal(ctx, tenantID, funnelID, columnID)

	algorithm := res.Algorithm
	if algorithm == "" {
		algorithm = "none"
	}
	outcome := string(res.Outcome)
	if err != nil {
		outcome = "error"
	}
	pickerTotal.WithLabelValues(algorithm, outcome).Inc()
	pickerDuration.WithLabelValues(algorithm).Observe(time.Since(start).Seconds())
	span.SetAttributes(
		attribute.String("algorithm", algorithm),
		attribute.String("outcome", outcome),
		attribute.String("picked_user_id", res.UserID),
	)
	return res, err
}

// pickInternal performs the actual picker logic. Kept private so PickForFunnel
// can wrap it with tracing + metrics without duplicating the algorithm.
func (p *LoadBalancePicker) pickInternal(ctx context.Context, tenantID, funnelID, columnID string) (funneldomain.PickResult, error) {
	// 1. Find all groups associated with this funnel that cover the column.
	groups, err := p.groupFunnelRepo.FindByFunnelID(ctx, funnelID)
	if err != nil {
		p.log.Error("load_balance.pick group_lookup_error",
			zap.String("tenant_id", tenantID),
			zap.String("funnel_id", funnelID),
			zap.Error(err),
		)
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
			p.log.Error("load_balance.pick lb_lookup_error",
				zap.String("tenant_id", tenantID),
				zap.String("group_id", gf.GroupID),
				zap.Error(err),
			)
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
		p.log.Error("load_balance.pick member_lookup_error",
			zap.String("tenant_id", tenantID),
			zap.String("group_id", chosen.groupID),
			zap.Error(err),
		)
		return p.fallbackToOwner(ctx, tenantID, "member_lookup_error")
	}
	members := make([]string, 0, len(ugs))
	for _, ug := range ugs {
		if _, err := p.userTenantRepo.FindByUserAndTenant(ctx, ug.UserID, tenantID); err != nil {
			if errors.Is(err, authdomain.ErrUserNotFound) {
				continue // legitimately not a tenant member
			}
			p.log.Error("load_balance.pick member_check_error",
				zap.String("tenant_id", tenantID),
				zap.String("user_id", ug.UserID),
				zap.Error(err),
			)
			return p.fallbackToOwner(ctx, tenantID, "member_check_error")
		}
		members = append(members, ug.UserID)
	}
	if len(members) == 0 {
		return p.fallbackToOwner(ctx, tenantID, "no_active_members")
	}

	// 4. Apply the algorithm.
	pickedUserID, err := p.applyAlgorithm(ctx, tenantID, chosen.cfg, members)
	if err != nil {
		p.log.Error("load_balance.pick algorithm_error",
			zap.String("tenant_id", tenantID),
			zap.String("group_id", chosen.groupID),
			zap.Error(err),
		)
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
		// Tiebreak: deterministic via the lexicographic sort applied to members.
		// A strictly-by-user.created_at tiebreak would require an extra query per pick;
		// stability is what matters for fairness over time, not the exact ordering key.
		counts, err := p.loadCounter.CountActiveByUsers(ctx, tenantID, members)
		if err != nil {
			return "", err
		}
		best := members[0]
		bestLoad := counts[best] // absent = 0
		for _, uid := range members[1:] {
			if counts[uid] < bestLoad {
				best, bestLoad = uid, counts[uid]
			}
		}
		return best, nil
	case authdomain.AlgorithmRandom:
		idx, err := cryptoRandIndex(len(members))
		if err != nil {
			return "", err
		}
		return members[idx], nil
	default:
		p.log.Warn("load_balance.pick unknown_algorithm",
			zap.String("tenant_id", tenantID),
			zap.String("algorithm", string(cfg.Algorithm)),
		)
		return members[0], nil
	}
}

func (p *LoadBalancePicker) fallbackToOwner(ctx context.Context, tenantID, reason string) (funneldomain.PickResult, error) {
	LoadBalanceFallbackTotal.WithLabelValues(reason).Inc()
	uts, err := p.userTenantRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return funneldomain.PickResult{}, err
	}
	for _, ut := range uts {
		if ut.IsOwner {
			fields := []zap.Field{
				zap.String("tenant_id", tenantID),
				zap.String("reason", reason),
				zap.String("picked_user_id", ut.UserID),
			}
			// Choose log level by reason. Infra-error reasons are Warn so the
			// fallback is noticed even without checking the preceding Error log.
			// multiple_active_groups is already logged at Error upstream; here we
			// stay at Info to avoid double-Error on the same event.
			switch reason {
			case "group_lookup_error", "lb_lookup_error", "member_lookup_error",
				"member_check_error", "algorithm_error":
				p.log.Warn("load_balance.pick fallback_owner", fields...)
			default:
				p.log.Info("load_balance.pick fallback_owner", fields...)
			}
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

// cryptoRandIndex returns a uniform random index in [0, n) using crypto/rand.
func cryptoRandIndex(n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("empty member list")
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}
