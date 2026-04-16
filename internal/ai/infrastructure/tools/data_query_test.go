package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeLeadSearcher struct {
	leads []funnelDomain.Lead
	err   error
}

func (f *fakeLeadSearcher) SearchByTenant(_ context.Context, _, _ string) ([]funnelDomain.Lead, error) {
	return f.leads, f.err
}

type fakeLeadDetailFetcher struct {
	lead  *funnelDomain.Lead
	notes []funnelDomain.LeadNote
	err   error
}

func (f *fakeLeadDetailFetcher) FindByID(_ context.Context, _, _ string) (*funnelDomain.Lead, error) {
	return f.lead, f.err
}

func (f *fakeLeadDetailFetcher) FindNotes(_ context.Context, _, _ string) ([]funnelDomain.LeadNote, error) {
	return f.notes, f.err
}

type fakeConversationHistoryFetcher struct {
	messages []HistoryMessage
	err      error
}

func (f *fakeConversationHistoryFetcher) FindMessages(_ context.Context, _ string, _ int) ([]HistoryMessage, error) {
	return f.messages, f.err
}

type fakeProductLister struct {
	products []productDomain.Product
	err      error
}

func (f *fakeProductLister) ListByTenant(_ context.Context, _ string) ([]productDomain.Product, error) {
	return f.products, f.err
}

type fakePipelineFetcher struct {
	columns []ColumnSummary
	err     error
}

func (f *fakePipelineFetcher) GetPipeline(_ context.Context, _, _ string) ([]ColumnSummary, error) {
	return f.columns, f.err
}

// ─── SearchLeadsTool tests ────────────────────────────────────────────────────

func TestSearchLeadsTool_Definition(t *testing.T) {
	tool := NewSearchLeadsTool(nil)
	def := tool.Definition()

	assert.Equal(t, "search_leads", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
	assert.Contains(t, def.Parameters, "query")
	assert.True(t, def.Parameters["query"].Required)
}

func TestSearchLeadsTool_Execute_WithResults(t *testing.T) {
	searcher := &fakeLeadSearcher{
		leads: []funnelDomain.Lead{
			{ID: "l1", Score: 80, Status: funnelDomain.LeadStatusOpen},
			{ID: "l2", Score: 50, Status: funnelDomain.LeadStatusWon},
		},
	}
	tool := NewSearchLeadsTool(searcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"query": "João"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "l1")
	assert.Contains(t, result.Content, "l2")
	assert.Contains(t, result.Content, "80")
}

func TestSearchLeadsTool_Execute_MissingQuery(t *testing.T) {
	tool := NewSearchLeadsTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "query")
}

func TestSearchLeadsTool_Execute_EmptyResults(t *testing.T) {
	searcher := &fakeLeadSearcher{leads: []funnelDomain.Lead{}}
	tool := NewSearchLeadsTool(searcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"query": "inexistente"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nenhum lead")
}

func TestSearchLeadsTool_Execute_RepositoryError(t *testing.T) {
	searcher := &fakeLeadSearcher{err: errors.New("db timeout")}
	tool := NewSearchLeadsTool(searcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"query": "test"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "db timeout")
}

// ─── GetLeadDetailTool tests ──────────────────────────────────────────────────

func TestGetLeadDetailTool_Definition(t *testing.T) {
	tool := NewGetLeadDetailTool(nil)
	def := tool.Definition()

	assert.Equal(t, "get_lead_detail", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.True(t, def.Parameters["lead_id"].Required)
}

func TestGetLeadDetailTool_Execute_WithNotes(t *testing.T) {
	lead := &funnelDomain.Lead{
		ID:             "l1",
		Status:         funnelDomain.LeadStatusOpen,
		Score:          70,
		ColumnID:       "col-1",
		ContactID:      "c-1",
		ConversationID: "conv-1",
		CreatedAt:      time.Now(),
	}
	notes := []funnelDomain.LeadNote{
		{ID: "n1", Content: "Primeiro contato", CreatedBy: "user-1", CreatedAt: time.Now()},
	}
	fetcher := &fakeLeadDetailFetcher{lead: lead, notes: notes}
	tool := NewGetLeadDetailTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"lead_id": "l1"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "l1")
	assert.Contains(t, result.Content, "70")
	assert.Contains(t, result.Content, "Primeiro contato")
}

func TestGetLeadDetailTool_Execute_WithoutNotes(t *testing.T) {
	lead := &funnelDomain.Lead{
		ID:             "l2",
		Status:         funnelDomain.LeadStatusWon,
		Score:          90,
		ColumnID:       "col-2",
		ContactID:      "c-2",
		ConversationID: "conv-2",
		CreatedAt:      time.Now(),
	}
	fetcher := &fakeLeadDetailFetcher{lead: lead, notes: []funnelDomain.LeadNote{}}
	tool := NewGetLeadDetailTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"lead_id": "l2"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nenhuma")
}

func TestGetLeadDetailTool_Execute_MissingLeadID(t *testing.T) {
	tool := NewGetLeadDetailTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "lead_id")
}

func TestGetLeadDetailTool_Execute_RepositoryError(t *testing.T) {
	fetcher := &fakeLeadDetailFetcher{err: errors.New("connection refused")}
	tool := NewGetLeadDetailTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"lead_id": "l1"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "connection refused")
}

func TestGetLeadDetailTool_Execute_LeadNotFound(t *testing.T) {
	fetcher := &fakeLeadDetailFetcher{lead: nil}
	tool := NewGetLeadDetailTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"lead_id": "missing"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nao encontrado")
}

// ─── GetConversationHistoryTool tests ────────────────────────────────────────

func TestGetConversationHistoryTool_Definition(t *testing.T) {
	tool := NewGetConversationHistoryTool(nil)
	def := tool.Definition()

	assert.Equal(t, "get_conversation_history", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
	assert.Contains(t, def.Parameters, "conversation_id")
	assert.True(t, def.Parameters["conversation_id"].Required)
	assert.Contains(t, def.Parameters, "limit")
	assert.False(t, def.Parameters["limit"].Required)
}

func TestGetConversationHistoryTool_Execute_WithMessages(t *testing.T) {
	now := time.Now()
	fetcher := &fakeConversationHistoryFetcher{
		messages: []HistoryMessage{
			{Role: "user", Content: "Olá, preciso de ajuda", Timestamp: now.Add(-2 * time.Minute)},
			{Role: "assistant", Content: "Claro, como posso ajudar?", Timestamp: now.Add(-1 * time.Minute)},
		},
	}
	tool := NewGetConversationHistoryTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"conversation_id": "conv-1",
		"limit":           float64(10),
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "conv-1")
	assert.Contains(t, result.Content, "Olá, preciso de ajuda")
	assert.Contains(t, result.Content, "user")
	assert.Contains(t, result.Content, "assistant")
}

func TestGetConversationHistoryTool_Execute_DefaultLimit(t *testing.T) {
	fetcher := &fakeConversationHistoryFetcher{
		messages: []HistoryMessage{
			{Role: "user", Content: "mensagem", Timestamp: time.Now()},
		},
	}
	tool := NewGetConversationHistoryTool(fetcher)

	// No limit provided — should default to 20
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"conversation_id": "conv-1",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestGetConversationHistoryTool_Execute_MissingConversationID(t *testing.T) {
	tool := NewGetConversationHistoryTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "conversation_id")
}

func TestGetConversationHistoryTool_Execute_EmptyHistory(t *testing.T) {
	fetcher := &fakeConversationHistoryFetcher{messages: []HistoryMessage{}}
	tool := NewGetConversationHistoryTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"conversation_id": "conv-empty",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nenhuma mensagem")
}

func TestGetConversationHistoryTool_Execute_RepositoryError(t *testing.T) {
	fetcher := &fakeConversationHistoryFetcher{err: errors.New("timeout")}
	tool := NewGetConversationHistoryTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"conversation_id": "conv-1",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "timeout")
}

// ─── ListProductsTool tests ───────────────────────────────────────────────────

func TestListProductsTool_Definition(t *testing.T) {
	tool := NewListProductsTool(nil)
	def := tool.Definition()

	assert.Equal(t, "list_products", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
}

func TestListProductsTool_Execute_WithProducts(t *testing.T) {
	lister := &fakeProductLister{
		products: []productDomain.Product{
			{ID: "p1", Name: "Consultoria Jurídica", Description: "Assessoria legal", Keywords: []string{"direito", "advogado"}},
			{ID: "p2", Name: "Peticionamento", Description: "Elaboração de peças", Keywords: []string{"petição"}},
		},
	}
	tool := NewListProductsTool(lister)

	result, err := tool.Execute(context.Background(), "tenant-1", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Consultoria Jurídica")
	assert.Contains(t, result.Content, "Peticionamento")
	assert.Contains(t, result.Content, "direito")
}

func TestListProductsTool_Execute_NoProducts(t *testing.T) {
	lister := &fakeProductLister{products: []productDomain.Product{}}
	tool := NewListProductsTool(lister)

	result, err := tool.Execute(context.Background(), "tenant-1", nil)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nenhum produto")
}

func TestListProductsTool_Execute_RepositoryError(t *testing.T) {
	lister := &fakeProductLister{err: errors.New("db error")}
	tool := NewListProductsTool(lister)

	result, err := tool.Execute(context.Background(), "tenant-1", nil)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "db error")
}

// ─── GetPipelineTool tests ────────────────────────────────────────────────────

func TestGetPipelineTool_Definition(t *testing.T) {
	tool := NewGetPipelineTool(nil)
	def := tool.Definition()

	assert.Equal(t, "get_pipeline", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
	assert.Contains(t, def.Parameters, "funnel_id")
	assert.False(t, def.Parameters["funnel_id"].Required)
}

func TestGetPipelineTool_Execute_WithColumns(t *testing.T) {
	fetcher := &fakePipelineFetcher{
		columns: []ColumnSummary{
			{ID: "col-1", Name: "Novo", LeadCount: 10},
			{ID: "col-2", Name: "Em Andamento", LeadCount: 5},
			{ID: "col-3", Name: "Fechado", LeadCount: 3},
		},
	}
	tool := NewGetPipelineTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"funnel_id": "f1"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "Novo")
	assert.Contains(t, result.Content, "Em Andamento")
	assert.Contains(t, result.Content, "10")
	assert.Contains(t, result.Content, "18") // total leads
}

func TestGetPipelineTool_Execute_DefaultFunnel(t *testing.T) {
	fetcher := &fakePipelineFetcher{
		columns: []ColumnSummary{
			{ID: "col-1", Name: "Entrada", LeadCount: 7},
		},
	}
	tool := NewGetPipelineTool(fetcher)

	// No funnel_id — should use default
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "padrão")
}

func TestGetPipelineTool_Execute_EmptyPipeline(t *testing.T) {
	fetcher := &fakePipelineFetcher{columns: []ColumnSummary{}}
	tool := NewGetPipelineTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "nenhuma coluna")
}

func TestGetPipelineTool_Execute_RepositoryError(t *testing.T) {
	fetcher := &fakePipelineFetcher{err: errors.New("db unavailable")}
	tool := NewGetPipelineTool(fetcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "db unavailable")
}
