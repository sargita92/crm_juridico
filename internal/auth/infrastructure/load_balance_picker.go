package infrastructure

import (
	"context"
	"errors"

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
	// Subsequent tasks will plug algorithm logic here. For now, always fall back.
	return p.fallbackToOwner(ctx, tenantID, "no_group")
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

// stub to appease "unused" linter for errors until algorithm tasks use it
var _ = errors.New
