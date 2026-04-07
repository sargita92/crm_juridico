package executors

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// AutoNoteExecutor saves an automatic note on a lead.
type AutoNoteExecutor struct {
	noteSaver domain.NoteSaver
}

func NewAutoNoteExecutor(n domain.NoteSaver) *AutoNoteExecutor {
	return &AutoNoteExecutor{noteSaver: n}
}

func (e *AutoNoteExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	template := a.ConfigString("template")
	if template == "" {
		template = "Automation executed"
	}
	return e.noteSaver.SaveNote(ctx, leadID, tenantID, template, "system")
}
