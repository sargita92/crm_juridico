package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// NoteCreator defines the contract for creating a note on a lead.
type NoteCreator interface {
	CreateNote(ctx context.Context, tenantID, leadID, content, createdBy string) error
}

// CreateLeadNoteTool adds a note to a lead on behalf of the AI specialist.
type CreateLeadNoteTool struct {
	creator NoteCreator
}

func NewCreateLeadNoteTool(creator NoteCreator) *CreateLeadNoteTool {
	return &CreateLeadNoteTool{creator: creator}
}

func (t *CreateLeadNoteTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"create_note",
		"Adiciona uma nota a um lead",
		domain.ToolCategoryCRMAction,
		map[string]domain.ParameterDef{
			"lead_id": {Type: "string", Description: "ID do lead", Required: true},
			"content": {Type: "string", Description: "Conteudo da nota", Required: true},
		},
	)
	return def
}

func (t *CreateLeadNoteTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	content, _ := args["content"].(string)
	if leadID == "" || content == "" {
		r := domain.NewToolResult("", "parameters obrigatorios: 'lead_id' e 'content'", true)
		return &r, nil
	}

	if err := t.creator.CreateNote(ctx, tenantID, leadID, content, "ai_specialist"); err != nil {
		r := domain.NewToolResult("", "erro ao criar nota: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Nota adicionada ao lead %s com sucesso", leadID), false)
	return &r, nil
}
