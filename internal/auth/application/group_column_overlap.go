package application

import (
	"context"
	"errors"
)

// GroupColumnOverlapChecker reports whether activating a load-balance config
// for a given group would create an overlap with another already-active group
// on the same funnel/column(s).
type GroupColumnOverlapChecker interface {
	// HasActiveOverlap returns (true, overlappingGroupIDs, nil) when another
	// group with an active LoadBalanceConfig already covers at least one of
	// the columns this group covers in the same tenant.
	HasActiveOverlap(ctx context.Context, tenantID, groupID string) (bool, []string, error)
}

// ErrActiveLoadBalanceOverlap is returned by ManageLoadBalanceUseCase.SetByGroup
// when another group with an active load-balance config already covers at least
// one of the columns the requested group covers (same tenant). Handlers should
// map this to 409 Conflict.
var ErrActiveLoadBalanceOverlap = errors.New("another group already has an active load balance covering the same funnel/column")
