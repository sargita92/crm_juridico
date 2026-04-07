package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// CreateAutomationInput holds the data required to create or update an automation.
type CreateAutomationInput struct {
	TenantID string
	FunnelID string
	ColumnID string
	Type     string
	Config   map[string]interface{}
	Priority int
}

// AutomationOutput is the DTO returned by CRUD operations.
type AutomationOutput struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id"`
	FunnelID string                 `json:"funnel_id"`
	ColumnID string                 `json:"column_id"`
	Type     string                 `json:"type"`
	Config   map[string]interface{} `json:"config"`
	Active   bool                   `json:"active"`
	Priority int                    `json:"priority"`
}

// LogOutput is the DTO returned when listing execution logs.
type LogOutput struct {
	ID           string    `json:"id"`
	AutomationID string    `json:"automation_id"`
	LeadID       string    `json:"lead_id"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	ExecutedAt   time.Time `json:"executed_at"`
}

// CRUDUseCase provides create/read/update/delete operations for Automation entities.
type CRUDUseCase struct {
	autoRepo domain.AutomationRepository
	logRepo  domain.ExecutionLogRepository
}

// NewCRUDUseCase constructs a CRUDUseCase.
func NewCRUDUseCase(autoRepo domain.AutomationRepository, logRepo domain.ExecutionLogRepository) *CRUDUseCase {
	return &CRUDUseCase{autoRepo: autoRepo, logRepo: logRepo}
}

// Create validates the input and persists a new Automation entity.
func (uc *CRUDUseCase) Create(ctx context.Context, input CreateAutomationInput) (*AutomationOutput, error) {
	auto, err := domain.NewAutomation(
		uuid.New().String(),
		input.TenantID,
		input.FunnelID,
		input.ColumnID,
		domain.AutomationType(input.Type),
		input.Config,
		input.Priority,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.autoRepo.Create(ctx, auto); err != nil {
		return nil, err
	}

	return toOutput(auto), nil
}

// Update loads, mutates and saves an existing Automation entity.
func (uc *CRUDUseCase) Update(ctx context.Context, id string, input CreateAutomationInput) (*AutomationOutput, error) {
	auto, err := uc.autoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	auto.Update(input.ColumnID, input.Config, input.Priority)

	if err := uc.autoRepo.Update(ctx, auto); err != nil {
		return nil, err
	}

	return toOutput(auto), nil
}

// Delete removes an Automation by ID.
func (uc *CRUDUseCase) Delete(ctx context.Context, id string) error {
	return uc.autoRepo.Delete(ctx, id)
}

// GetByID returns the Automation with the given ID.
func (uc *CRUDUseCase) GetByID(ctx context.Context, id string) (*AutomationOutput, error) {
	auto, err := uc.autoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toOutput(auto), nil
}

// ListByFunnel returns all automations for the given tenant and funnel.
func (uc *CRUDUseCase) ListByFunnel(ctx context.Context, tenantID, funnelID string) ([]AutomationOutput, error) {
	automations, err := uc.autoRepo.FindByFunnelID(ctx, tenantID, funnelID)
	if err != nil {
		return nil, err
	}

	out := make([]AutomationOutput, len(automations))
	for i, a := range automations {
		cp := a
		out[i] = *toOutput(&cp)
	}
	return out, nil
}

// Toggle activates or deactivates an Automation depending on its current state.
func (uc *CRUDUseCase) Toggle(ctx context.Context, id string) error {
	auto, err := uc.autoRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if auto.Active {
		auto.Deactivate()
	} else {
		auto.Activate()
	}

	return uc.autoRepo.Update(ctx, auto)
}

// GetLogs returns paginated execution logs for the given automation.
func (uc *CRUDUseCase) GetLogs(ctx context.Context, automationID string, limit, offset int) ([]LogOutput, error) {
	logs, err := uc.logRepo.FindByAutomationID(ctx, automationID, limit, offset)
	if err != nil {
		return nil, err
	}

	out := make([]LogOutput, len(logs))
	for i, l := range logs {
		out[i] = LogOutput{
			ID:           l.ID,
			AutomationID: l.AutomationID,
			LeadID:       l.LeadID,
			Status:       string(l.Status),
			ErrorMessage: l.ErrorMessage,
			ExecutedAt:   l.ExecutedAt,
		}
	}
	return out, nil
}

func toOutput(a *domain.Automation) *AutomationOutput {
	return &AutomationOutput{
		ID:       a.ID,
		TenantID: a.TenantID,
		FunnelID: a.FunnelID,
		ColumnID: a.ColumnID,
		Type:     string(a.Type),
		Config:   a.Config,
		Active:   a.Active,
		Priority: a.Priority,
	}
}
