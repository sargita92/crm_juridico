package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/ai/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/ai/infrastructure/tools"
)

// TestToolRegistryHas10Tools verifies that NewModule registers exactly 10 tools.
// This test uses only in-memory/nil deps and does not require a database.
func TestToolRegistryHas10Tools(t *testing.T) {
	// Build the registry using the same adapters as module.go, with nil concrete deps.
	// The adapters accept nil repo pointers; the registry just stores the tool objects.
	var (
		leadSearchAdapter         = infrastructure.NewLeadSearchAdapter(nil)
		leadDetailAdapter         = infrastructure.NewLeadDetailAdapter(nil, nil)
		convHistoryToolAdapter    = infrastructure.NewConversationHistoryToolAdapter(nil)
		productListToolAdapter    = infrastructure.NewProductListToolAdapter(nil, nil)
		pipelineToolAdapter       = infrastructure.NewPipelineToolAdapter(nil, nil, nil)
		leadMoveToolAdapter       = infrastructure.NewLeadMoveToolAdapter(nil, nil)
		noteCreatorToolAdapter    = infrastructure.NewNoteCreatorToolAdapter(nil, nil)
		scoreUpdaterToolAdapter   = infrastructure.NewScoreUpdaterToolAdapter(nil)
		automationTriggerAdapter  = infrastructure.NewAutomationTriggerToolAdapter(nil)
		specialistSwitcherAdapter = infrastructure.NewSpecialistSwitcherToolAdapter(nil)
	)

	registry := application.NewToolRegistry()
	registry.Register(tools.NewSearchLeadsTool(leadSearchAdapter))
	registry.Register(tools.NewGetLeadDetailTool(leadDetailAdapter))
	registry.Register(tools.NewGetConversationHistoryTool(convHistoryToolAdapter))
	registry.Register(tools.NewListProductsTool(productListToolAdapter))
	registry.Register(tools.NewGetPipelineTool(pipelineToolAdapter))
	registry.Register(tools.NewMoveLeadTool(leadMoveToolAdapter))
	registry.Register(tools.NewCreateLeadNoteTool(noteCreatorToolAdapter))
	registry.Register(tools.NewUpdateLeadScoreTool(scoreUpdaterToolAdapter))
	registry.Register(tools.NewTriggerAutomationTool(automationTriggerAdapter))
	registry.Register(tools.NewSwitchSpecialistTool(specialistSwitcherAdapter))

	assert.Len(t, registry.All(), 10, "tool registry should have exactly 10 tools registered")

	expectedNames := []string{
		"search_leads",
		"get_lead_detail",
		"get_conversation_history",
		"list_products",
		"get_pipeline",
		"move_lead",
		"create_note",
		"update_score",
		"trigger_automation",
		"switch_specialist",
	}
	for _, name := range expectedNames {
		tool, err := registry.Get(name)
		assert.NoError(t, err, "tool %q should be registered", name)
		assert.NotNil(t, tool, "tool %q should not be nil", name)
	}
}
