package tools

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// AutomationTrigger defines the contract for triggering an automation manually.
type AutomationTrigger interface {
	TriggerManually(ctx context.Context, tenantID, automationID, leadID string) (string, error)
}

// TriggerAutomationTool triggers an automation manually for a given lead.
type TriggerAutomationTool struct {
	trigger AutomationTrigger
}

func NewTriggerAutomationTool(trigger AutomationTrigger) *TriggerAutomationTool {
	return &TriggerAutomationTool{trigger: trigger}
}

func (t *TriggerAutomationTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"trigger_automation",
		"Dispara uma automacao manualmente para um lead",
		domain.ToolCategoryAutomation,
		map[string]domain.ParameterDef{
			"automation_id": {Type: "string", Description: "ID da automacao", Required: true},
			"lead_id":       {Type: "string", Description: "ID do lead alvo", Required: true},
		},
	)
	return def
}

func (t *TriggerAutomationTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	automationID, _ := args["automation_id"].(string)
	leadID, _ := args["lead_id"].(string)
	if automationID == "" || leadID == "" {
		r := domain.NewToolResult("", "parametros obrigatorios: 'automation_id' e 'lead_id'", true)
		return &r, nil
	}

	output, err := t.trigger.TriggerManually(ctx, tenantID, automationID, leadID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao disparar automacao: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", output, false)
	return &r, nil
}
