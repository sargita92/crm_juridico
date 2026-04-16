package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// LeadDetailFetcher defines the contract for fetching lead details and notes.
type LeadDetailFetcher interface {
	FindByID(ctx context.Context, tenantID, leadID string) (*funnelDomain.Lead, error)
	FindNotes(ctx context.Context, tenantID, leadID string) ([]funnelDomain.LeadNote, error)
}

// GetLeadDetailTool returns full details of a lead including its notes.
type GetLeadDetailTool struct {
	fetcher LeadDetailFetcher
}

func NewGetLeadDetailTool(fetcher LeadDetailFetcher) *GetLeadDetailTool {
	return &GetLeadDetailTool{fetcher: fetcher}
}

func (t *GetLeadDetailTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"get_lead_detail",
		"Retorna detalhes completos de um lead incluindo notas",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"lead_id": {
				Type:        "string",
				Description: "ID do lead",
				Required:    true,
			},
		},
	)
	return def
}

func (t *GetLeadDetailTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	if leadID == "" {
		r := domain.NewToolResult("", "parametro obrigatorio 'lead_id' nao informado", true)
		return &r, nil
	}

	lead, err := t.fetcher.FindByID(ctx, tenantID, leadID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao buscar lead: "+err.Error(), true)
		return &r, nil
	}
	if lead == nil {
		r := domain.NewToolResult("", "lead nao encontrado: "+leadID, false)
		return &r, nil
	}

	notes, err := t.fetcher.FindNotes(ctx, tenantID, leadID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao buscar notas do lead: "+err.Error(), true)
		return &r, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Lead ID: %s\n", lead.ID))
	sb.WriteString(fmt.Sprintf("Status: %s\n", lead.Status))
	sb.WriteString(fmt.Sprintf("Score: %d\n", lead.Score))
	sb.WriteString(fmt.Sprintf("Column ID: %s\n", lead.ColumnID))
	sb.WriteString(fmt.Sprintf("Contact ID: %s\n", lead.ContactID))
	sb.WriteString(fmt.Sprintf("Conversation ID: %s\n", lead.ConversationID))
	if lead.ProductID != "" {
		sb.WriteString(fmt.Sprintf("Product ID: %s\n", lead.ProductID))
	}
	if lead.ResponsibleUserID != "" {
		sb.WriteString(fmt.Sprintf("Responsible: %s\n", lead.ResponsibleUserID))
	}
	sb.WriteString(fmt.Sprintf("Created At: %s\n", lead.CreatedAt.Format("2006-01-02 15:04:05")))

	if len(notes) == 0 {
		sb.WriteString("Notas: nenhuma\n")
	} else {
		sb.WriteString(fmt.Sprintf("Notas (%d):\n", len(notes)))
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n",
				note.CreatedAt.Format("2006-01-02 15:04"),
				note.CreatedBy,
				note.Content,
			))
		}
	}

	r := domain.NewToolResult("", sb.String(), false)
	return &r, nil
}
