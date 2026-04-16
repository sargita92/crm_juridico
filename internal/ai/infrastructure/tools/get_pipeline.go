package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// ColumnSummary holds the essential info about a pipeline column.
type ColumnSummary struct {
	ID        string
	Name      string
	LeadCount int
}

// PipelineFetcher defines the contract for fetching pipeline columns.
type PipelineFetcher interface {
	GetPipeline(ctx context.Context, tenantID, funnelID string) ([]ColumnSummary, error)
}

// GetPipelineTool retrieves the columns and lead counts of a funnel pipeline.
type GetPipelineTool struct {
	fetcher PipelineFetcher
}

func NewGetPipelineTool(fetcher PipelineFetcher) *GetPipelineTool {
	return &GetPipelineTool{fetcher: fetcher}
}

func (t *GetPipelineTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"get_pipeline",
		"Retorna as colunas e contagem de leads do funil",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"funnel_id": {
				Type:        "string",
				Description: "ID do funil (se vazio, usa o funil padrão do tenant)",
				Required:    false,
			},
		},
	)
	return def
}

func (t *GetPipelineTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	funnelID, _ := args["funnel_id"].(string)

	columns, err := t.fetcher.GetPipeline(ctx, tenantID, funnelID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao buscar pipeline: "+err.Error(), true)
		return &r, nil
	}

	if len(columns) == 0 {
		r := domain.NewToolResult("", "nenhuma coluna encontrada no funil", false)
		return &r, nil
	}

	var sb strings.Builder
	if funnelID == "" {
		sb.WriteString("Pipeline do funil padrão:\n")
	} else {
		sb.WriteString(fmt.Sprintf("Pipeline do funil %s:\n", funnelID))
	}

	totalLeads := 0
	for _, col := range columns {
		sb.WriteString(fmt.Sprintf("- %s: %d leads\n", col.Name, col.LeadCount))
		totalLeads += col.LeadCount
	}
	sb.WriteString(fmt.Sprintf("Total: %d leads em %d colunas\n", totalLeads, len(columns)))

	r := domain.NewToolResult("", strings.TrimRight(sb.String(), "\n"), false)
	return &r, nil
}
