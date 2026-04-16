package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
)

// ProductLister defines the contract for listing products by tenant.
type ProductLister interface {
	ListByTenant(ctx context.Context, tenantID string) ([]productDomain.Product, error)
}

// ListProductsTool returns the list of products available for the tenant.
type ListProductsTool struct {
	lister ProductLister
}

func NewListProductsTool(lister ProductLister) *ListProductsTool {
	return &ListProductsTool{lister: lister}
}

func (t *ListProductsTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"list_products",
		"Lista os produtos disponíveis para o tenant",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{},
	)
	return def
}

func (t *ListProductsTool) Execute(ctx context.Context, tenantID string, _ map[string]interface{}) (*domain.ToolResult, error) {
	products, err := t.lister.ListByTenant(ctx, tenantID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao listar produtos: "+err.Error(), true)
		return &r, nil
	}

	if len(products) == 0 {
		r := domain.NewToolResult("", "nenhum produto cadastrado para o tenant", false)
		return &r, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Produtos disponíveis (%d):\n", len(products)))
	for _, p := range products {
		sb.WriteString(fmt.Sprintf("- %s: %s", p.Name, p.Description))
		if len(p.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf(" [palavras-chave: %s]", strings.Join(p.Keywords, ", ")))
		}
		sb.WriteString("\n")
	}

	r := domain.NewToolResult("", sb.String(), false)
	return &r, nil
}
