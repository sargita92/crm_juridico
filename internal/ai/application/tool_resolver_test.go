package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSpecialistToolFinder struct {
	toolNames map[string][]string
}

func (f *fakeSpecialistToolFinder) FindToolNamesBySpecialistID(_ context.Context, specialistID string) ([]string, error) {
	return f.toolNames[specialistID], nil
}

func setupResolver() (*ToolResolver, *ToolRegistry) {
	reg := NewToolRegistry()
	reg.Register(newFakeTool("search_leads", domain.ToolCategoryDataQuery))
	reg.Register(newFakeTool("move_lead", domain.ToolCategoryCRMAction))
	reg.Register(newFakeTool("trigger_automation", domain.ToolCategoryAutomation))

	finder := &fakeSpecialistToolFinder{
		toolNames: map[string][]string{
			"spec-1": {"search_leads", "move_lead"},
			"spec-2": {"search_leads"},
		},
	}

	return NewToolResolver(reg, finder), reg
}

func TestToolResolver_ResolveForSpecialist_ReturnsAssociatedTools(t *testing.T) {
	resolver, _ := setupResolver()
	ctx := context.Background()

	tools, err := resolver.ResolveForSpecialist(ctx, "spec-1")
	require.NoError(t, err)
	assert.Len(t, tools, 2)

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Definition().Name
	}
	assert.Contains(t, names, "search_leads")
	assert.Contains(t, names, "move_lead")
}

func TestToolResolver_ResolveForSpecialist_NoAssociations_ReturnsEmpty(t *testing.T) {
	resolver, _ := setupResolver()
	tools, err := resolver.ResolveForSpecialist(context.Background(), "spec-unknown")
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestToolResolver_ApplyStepConstraints_WithForcedTools_FiltersToForcedOnly(t *testing.T) {
	resolver, _ := setupResolver()

	tools := []domain.Tool{
		newFakeTool("search_leads", domain.ToolCategoryDataQuery),
		newFakeTool("move_lead", domain.ToolCategoryCRMAction),
	}

	step := &specDomain.Step{ForcedTools: []string{"move_lead"}}
	filtered := resolver.ApplyStepConstraints(tools, step)

	assert.Len(t, filtered, 1)
	assert.Equal(t, "move_lead", filtered[0].Definition().Name)
}

func TestToolResolver_ApplyStepConstraints_WithRestrictedTools_RemovesRestricted(t *testing.T) {
	resolver, _ := setupResolver()

	tools := []domain.Tool{
		newFakeTool("search_leads", domain.ToolCategoryDataQuery),
		newFakeTool("move_lead", domain.ToolCategoryCRMAction),
	}

	step := &specDomain.Step{RestrictedTools: []string{"move_lead"}}
	filtered := resolver.ApplyStepConstraints(tools, step)

	assert.Len(t, filtered, 1)
	assert.Equal(t, "search_leads", filtered[0].Definition().Name)
}

func TestToolResolver_ApplyStepConstraints_NilStep_ReturnsAllTools(t *testing.T) {
	resolver, _ := setupResolver()

	tools := []domain.Tool{
		newFakeTool("search_leads", domain.ToolCategoryDataQuery),
	}

	filtered := resolver.ApplyStepConstraints(tools, nil)
	assert.Len(t, filtered, 1)
}

func TestToolResolver_ResolveDefinitions_FiltersAndReturnsDefinitions(t *testing.T) {
	resolver, _ := setupResolver()
	ctx := context.Background()

	step := &specDomain.Step{RestrictedTools: []string{"move_lead"}}
	defs, err := resolver.ResolveDefinitions(ctx, "spec-1", step)
	require.NoError(t, err)
	assert.Len(t, defs, 1)
	assert.Equal(t, "search_leads", defs[0].Name)
}
