package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// LeadSearcher defines the contract for searching leads by tenant and query.
type LeadSearcher interface {
	SearchByTenant(ctx context.Context, tenantID, query string) ([]funnelDomain.Lead, error)
}

// SearchLeadsTool searches leads by name, phone, or status.
type SearchLeadsTool struct {
	searcher LeadSearcher
}

func NewSearchLeadsTool(searcher LeadSearcher) *SearchLeadsTool {
	return &SearchLeadsTool{searcher: searcher}
}

func (t *SearchLeadsTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"search_leads",
		"Busca leads por nome, telefone ou status",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"query": {
				Type:        "string",
				Description: "Termo de busca (nome, telefone ou status)",
				Required:    true,
			},
		},
	)
	return def
}

func (t *SearchLeadsTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	query, _ := args["query"].(string)
	if query == "" {
		r := domain.NewToolResult("", "parametro obrigatorio 'query' nao informado", true)
		return &r, nil
	}

	leads, err := t.searcher.SearchByTenant(ctx, tenantID, query)
	if err != nil {
		r := domain.NewToolResult("", "erro ao buscar leads: "+err.Error(), true)
		return &r, nil
	}

	if len(leads) == 0 {
		r := domain.NewToolResult("", "nenhum lead encontrado para: "+query, false)
		return &r, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Encontrados %d leads:\n", len(leads)))
	for _, lead := range leads {
		sb.WriteString(fmt.Sprintf("- ID: %s | Score: %d | Status: %s\n", lead.ID, lead.Score, lead.Status))
	}

	r := domain.NewToolResult("", sb.String(), false)
	return &r, nil
}
