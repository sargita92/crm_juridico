package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// ScoreUpdater defines the contract for updating a lead's qualification score.
type ScoreUpdater interface {
	UpdateScore(ctx context.Context, tenantID, leadID string, score int) error
}

// UpdateLeadScoreTool updates the qualification score of a lead.
type UpdateLeadScoreTool struct {
	updater ScoreUpdater
}

func NewUpdateLeadScoreTool(updater ScoreUpdater) *UpdateLeadScoreTool {
	return &UpdateLeadScoreTool{updater: updater}
}

func (t *UpdateLeadScoreTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"update_score",
		"Atualiza o score de qualificacao de um lead",
		domain.ToolCategoryCRMAction,
		map[string]domain.ParameterDef{
			"lead_id": {Type: "string", Description: "ID do lead", Required: true},
			"score":   {Type: "number", Description: "Novo score de qualificacao (0-100)", Required: true},
		},
	)
	return def
}

func (t *UpdateLeadScoreTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	if leadID == "" {
		r := domain.NewToolResult("", "parametro obrigatorio 'lead_id' nao informado", true)
		return &r, nil
	}

	var score int
	switch v := args["score"].(type) {
	case float64:
		score = int(v)
	case int:
		score = v
	default:
		r := domain.NewToolResult("", "parametro 'score' deve ser um numero", true)
		return &r, nil
	}

	if err := t.updater.UpdateScore(ctx, tenantID, leadID, score); err != nil {
		r := domain.NewToolResult("", "erro ao atualizar score: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Score do lead %s atualizado para %d", leadID, score), false)
	return &r, nil
}
