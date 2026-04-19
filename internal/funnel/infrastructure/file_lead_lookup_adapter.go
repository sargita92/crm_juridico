package infrastructure

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// FileLeadLookupAdapter implements the files domain's LeadLookup port by
// delegating to the funnel LeadRepository. It filters out leads that belong
// to a different tenant — the files module trusts this isolation.
type FileLeadLookupAdapter struct {
	repo domain.LeadRepository
}

func NewFileLeadLookupAdapter(repo domain.LeadRepository) *FileLeadLookupAdapter {
	return &FileLeadLookupAdapter{repo: repo}
}

func (a *FileLeadLookupAdapter) FindByConversation(ctx context.Context, tenantID, conversationID string) (string, bool, error) {
	lead, err := a.repo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, domain.ErrLeadNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	if lead.TenantID != tenantID {
		return "", false, nil
	}
	return lead.ID, true, nil
}
