package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type AssignLeadInput struct {
	LeadID   string
	UserID   string
	TenantID string
}

type AssignLeadUseCase struct {
	leadRepo domain.LeadRepository
}

func NewAssignLeadUseCase(leadRepo domain.LeadRepository) *AssignLeadUseCase {
	return &AssignLeadUseCase{leadRepo: leadRepo}
}

func (uc *AssignLeadUseCase) Execute(ctx context.Context, input AssignLeadInput) error {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return err
	}
	if lead.TenantID != input.TenantID {
		return domain.ErrLeadNotFound
	}
	lead.AssignResponsible(input.UserID)
	return uc.leadRepo.Update(ctx, lead)
}
