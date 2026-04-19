package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type LeadFilesSummaryOutput struct {
	Total  int64
	Recent []domain.File
}

type LeadFilesSummaryUseCase struct {
	repo  domain.FileRepository
	limit int
}

func NewLeadFilesSummaryUseCase(repo domain.FileRepository) *LeadFilesSummaryUseCase {
	return &LeadFilesSummaryUseCase{repo: repo, limit: 6}
}

func (uc *LeadFilesSummaryUseCase) Execute(ctx context.Context, tenantID, leadID string) (*LeadFilesSummaryOutput, error) {
	if tenantID == "" {
		return nil, domain.ErrTenantIDRequired
	}
	total, err := uc.repo.CountByLead(ctx, tenantID, leadID)
	if err != nil {
		return nil, err
	}
	recent, err := uc.repo.ListRecentByLead(ctx, tenantID, leadID, uc.limit)
	if err != nil {
		return nil, err
	}
	return &LeadFilesSummaryOutput{Total: total, Recent: recent}, nil
}
