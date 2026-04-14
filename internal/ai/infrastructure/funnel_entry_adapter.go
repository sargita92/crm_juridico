package infrastructure

import (
	"context"
	"errors"
	"fmt"

	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// FunnelEntryAdapter implements application.EntryColumnFinder using the funnel
// repositories. It returns an empty string (and nil error) when the tenant has
// no default funnel or that funnel has no entry column — the reset usecase
// treats that as "skip the lead move, but still confirm the reset".
type FunnelEntryAdapter struct {
	funnelRepo funnelDomain.FunnelRepository
	colRepo    funnelDomain.ColumnRepository
}

func NewFunnelEntryAdapter(funnelRepo funnelDomain.FunnelRepository, colRepo funnelDomain.ColumnRepository) *FunnelEntryAdapter {
	return &FunnelEntryAdapter{funnelRepo: funnelRepo, colRepo: colRepo}
}

// FindEntryColumnID returns the ID of the entry column of the tenant's default
// funnel. Returns "" if the tenant has no default funnel or the funnel has no
// entry column (both are normal, not errors from the caller's perspective).
func (a *FunnelEntryAdapter) FindEntryColumnID(ctx context.Context, tenantID string) (string, error) {
	f, err := a.funnelRepo.FindDefaultByTenantID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, funnelDomain.ErrFunnelNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("funnel_entry_adapter: find default funnel: %w", err)
	}
	if f == nil {
		return "", nil
	}
	col, err := a.colRepo.FindEntryByFunnelID(ctx, f.ID)
	if err != nil {
		if errors.Is(err, funnelDomain.ErrColumnNotFound) {
			return "", nil
		}
		return "", fmt.Errorf("funnel_entry_adapter: find entry column: %w", err)
	}
	if col == nil {
		return "", nil
	}
	return col.ID, nil
}

// LeadResetterAdapter implements application.LeadResetter. It looks up the
// lead tied to a conversation and moves it back to the entry column, zeroing
// its score. Missing leads are treated as a no-op: the reset usecase's
// semantics say the reset proceeds anyway even if there is no lead yet.
type LeadResetterAdapter struct {
	leadRepo funnelDomain.LeadRepository
}

func NewLeadResetterAdapter(leadRepo funnelDomain.LeadRepository) *LeadResetterAdapter {
	return &LeadResetterAdapter{leadRepo: leadRepo}
}

func (a *LeadResetterAdapter) ResetLeadForConversation(ctx context.Context, conversationID, entryColumnID string) error {
	lead, err := a.leadRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, funnelDomain.ErrLeadNotFound) {
			return nil
		}
		return fmt.Errorf("lead_resetter: find lead: %w", err)
	}
	if lead == nil {
		return nil
	}
	lead.ColumnID = entryColumnID
	lead.Score = 0
	if err := a.leadRepo.Update(ctx, lead); err != nil {
		return fmt.Errorf("lead_resetter: update lead: %w", err)
	}
	return nil
}
