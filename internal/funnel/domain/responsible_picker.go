package domain

import (
	"context"
	"errors"
)

// PickOutcome classifies how the responsible user was chosen.
type PickOutcome string

const (
	PickOutcomePicked        PickOutcome = "picked"
	PickOutcomeFallbackOwner PickOutcome = "fallback_owner"
)

// PickResult is the outcome of a single responsible-user selection.
type PickResult struct {
	UserID    string
	Algorithm string // empty when Outcome == PickOutcomeFallbackOwner
	GroupID   string // empty when Outcome == PickOutcomeFallbackOwner
	Outcome   PickOutcome
}

// ErrNoResponsibleAvailable means neither the load-balance flow nor the tenant
// owner fallback could produce a user. Lead creation MUST abort.
var ErrNoResponsibleAvailable = errors.New("no responsible user available for tenant")

// ResponsiblePicker chooses a user to receive a newly created lead.
//
// Implementations must never return an empty UserID on a nil error: if the
// load-balance flow fails, they MUST resolve the tenant owner. When neither
// is possible, they return ErrNoResponsibleAvailable.
type ResponsiblePicker interface {
	PickForFunnel(ctx context.Context, tenantID, funnelID, columnID string) (PickResult, error)
}

// LeadLoadCounter is a read-only port used by picker implementations to
// evaluate the "least load" algorithm. It counts currently-open leads
// assigned to each user in the supplied list, scoped to a single tenant.
type LeadLoadCounter interface {
	CountActiveByUsers(ctx context.Context, tenantID string, userIDs []string) (map[string]int, error)
}
