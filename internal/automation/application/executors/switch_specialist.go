package executors

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// SwitchSpecialistExecutor reassigns the conversation of a lead to a different specialist.
type SwitchSpecialistExecutor struct {
	switcher   domain.SpecialistSwitcher
	leadFinder domain.LeadFinder
}

func NewSwitchSpecialistExecutor(s domain.SpecialistSwitcher, f domain.LeadFinder) *SwitchSpecialistExecutor {
	return &SwitchSpecialistExecutor{switcher: s, leadFinder: f}
}

func (e *SwitchSpecialistExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil {
		return err
	}
	return e.switcher.SwitchSpecialist(ctx, lead.ConversationID, a.ConfigString("specialist_id"))
}
