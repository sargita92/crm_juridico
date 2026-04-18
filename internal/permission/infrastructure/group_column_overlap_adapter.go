package infrastructure

import (
	"context"
	"errors"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	permdomain "github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// GroupColumnOverlapAdapter implements auth/application.GroupColumnOverlapChecker
// by combining the permission's GroupFunnelRepository (to discover funnel/column
// coverage) with the auth's LoadBalanceConfigRepository (to check which peers
// have an active load-balance config).
type GroupColumnOverlapAdapter struct {
	groupFunnelRepo permdomain.GroupFunnelRepository
	lbRepo          authdomain.LoadBalanceConfigRepository
}

// NewGroupColumnOverlapAdapter builds the adapter.
func NewGroupColumnOverlapAdapter(
	gfr permdomain.GroupFunnelRepository,
	lbr authdomain.LoadBalanceConfigRepository,
) *GroupColumnOverlapAdapter {
	return &GroupColumnOverlapAdapter{groupFunnelRepo: gfr, lbRepo: lbr}
}

// HasActiveOverlap returns (true, peerGroupIDs, nil) when another group within
// the same tenant has an active LoadBalanceConfig and its funnel/column
// coverage overlaps with the requesting group's coverage.
func (a *GroupColumnOverlapAdapter) HasActiveOverlap(ctx context.Context, tenantID, groupID string) (bool, []string, error) {
	// 1. Coverage of the requesting group.
	mine, err := a.groupFunnelRepo.FindByGroupID(ctx, groupID)
	if err != nil {
		return false, nil, err
	}
	if len(mine) == 0 {
		return false, nil, nil
	}

	overlapping := map[string]struct{}{}
	for _, gf := range mine {
		// 2. Peers on the same funnel.
		peers, err := a.groupFunnelRepo.FindByFunnelID(ctx, gf.FunnelID)
		if err != nil {
			return false, nil, err
		}
		for _, peer := range peers {
			if peer.GroupID == groupID {
				continue
			}
			if !columnsOverlap(gf, peer) {
				continue
			}
			// 3. Does the peer have an ACTIVE load-balance config in this tenant?
			cfg, err := a.lbRepo.FindByGroupID(ctx, tenantID, peer.GroupID)
			if err != nil {
				if errors.Is(err, authdomain.ErrLoadBalanceNotFound) {
					continue
				}
				return false, nil, err
			}
			if cfg != nil && cfg.Active {
				overlapping[peer.GroupID] = struct{}{}
			}
		}
	}

	if len(overlapping) == 0 {
		return false, nil, nil
	}
	out := make([]string, 0, len(overlapping))
	for id := range overlapping {
		out = append(out, id)
	}
	return true, out, nil
}

// columnsOverlap reports whether two GroupFunnel associations cover at least
// one common column on the same funnel. An empty ColumnIDs slice means the
// group covers the whole funnel, so the overlap is unconditional on that side.
func columnsOverlap(a, b permdomain.GroupFunnel) bool {
	if len(a.ColumnIDs) == 0 || len(b.ColumnIDs) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(a.ColumnIDs))
	for _, id := range a.ColumnIDs {
		set[id] = struct{}{}
	}
	for _, id := range b.ColumnIDs {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}
