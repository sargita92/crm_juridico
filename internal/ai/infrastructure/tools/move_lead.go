package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// LeadMover defines the contract for moving a lead to another kanban column.
type LeadMover interface {
	MoveToColumn(ctx context.Context, tenantID, leadID, columnID string) error
}

// MoveLeadTool moves a lead to a different kanban column.
type MoveLeadTool struct {
	mover LeadMover
}

func NewMoveLeadTool(mover LeadMover) *MoveLeadTool {
	return &MoveLeadTool{mover: mover}
}

func (t *MoveLeadTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"move_lead",
		"Move um lead para outra coluna do kanban",
		domain.ToolCategoryCRMAction,
		map[string]domain.ParameterDef{
			"lead_id":   {Type: "string", Description: "ID do lead", Required: true},
			"column_id": {Type: "string", Description: "ID da coluna destino", Required: true},
		},
	)
	return def
}

func (t *MoveLeadTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	columnID, _ := args["column_id"].(string)
	if leadID == "" || columnID == "" {
		r := domain.NewToolResult("", "parameters obrigatorios: 'lead_id' e 'column_id'", true)
		return &r, nil
	}

	if err := t.mover.MoveToColumn(ctx, tenantID, leadID, columnID); err != nil {
		r := domain.NewToolResult("", "erro ao mover lead: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Lead %s movido para coluna %s com sucesso", leadID, columnID), false)
	return &r, nil
}
