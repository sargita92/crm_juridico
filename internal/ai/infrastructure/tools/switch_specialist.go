package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// SpecialistSwitcher defines the contract for switching the specialist on a conversation.
type SpecialistSwitcher interface {
	SwitchSpecialist(ctx context.Context, tenantID, conversationID, specialistID string) error
}

// SwitchSpecialistTool switches the AI specialist assigned to a conversation.
type SwitchSpecialistTool struct {
	switcher SpecialistSwitcher
}

func NewSwitchSpecialistTool(switcher SpecialistSwitcher) *SwitchSpecialistTool {
	return &SwitchSpecialistTool{switcher: switcher}
}

func (t *SwitchSpecialistTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"switch_specialist",
		"Troca o especialista IA responsavel por uma conversa",
		domain.ToolCategoryAutomation,
		map[string]domain.ParameterDef{
			"specialist_id":   {Type: "string", Description: "ID do especialista destino", Required: true},
			"conversation_id": {Type: "string", Description: "ID da conversa onde a troca sera aplicada", Required: true},
		},
	)
	return def
}

func (t *SwitchSpecialistTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	specialistID, _ := args["specialist_id"].(string)
	conversationID, _ := args["conversation_id"].(string)
	if specialistID == "" || conversationID == "" {
		r := domain.NewToolResult("", "parameters obrigatorios: 'specialist_id' e 'conversation_id'", true)
		return &r, nil
	}

	if err := t.switcher.SwitchSpecialist(ctx, tenantID, conversationID, specialistID); err != nil {
		r := domain.NewToolResult("", "erro ao trocar especialista: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Especialista trocado para %s na conversa %s com sucesso", specialistID, conversationID), false)
	return &r, nil
}
