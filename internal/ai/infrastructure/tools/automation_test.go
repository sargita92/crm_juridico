package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── fakes ───────────────────────────────────────────────────────────────────

type fakeAutomationTrigger struct {
	output string
	err    error
}

func (f *fakeAutomationTrigger) TriggerManually(_ context.Context, _, _, _ string) (string, error) {
	return f.output, f.err
}

type fakeSpecialistSwitcher struct {
	err error
}

func (f *fakeSpecialistSwitcher) SwitchSpecialist(_ context.Context, _, _, _ string) error {
	return f.err
}

// ─── TriggerAutomationTool tests ──────────────────────────────────────────────

func TestTriggerAutomationTool_Definition(t *testing.T) {
	tool := NewTriggerAutomationTool(nil)
	def := tool.Definition()

	assert.Equal(t, "trigger_automation", def.Name)
	assert.Equal(t, domain.ToolCategoryAutomation, def.Category)
	assert.Contains(t, def.Parameters, "automation_id")
	assert.True(t, def.Parameters["automation_id"].Required)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.True(t, def.Parameters["lead_id"].Required)
}

func TestTriggerAutomationTool_Execute_HappyPath(t *testing.T) {
	trigger := &fakeAutomationTrigger{output: "automacao executada com 3 acoes"}
	tool := NewTriggerAutomationTool(trigger)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"automation_id": "auto-99",
		"lead_id":       "lead-42",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "automacao executada com 3 acoes")
}

func TestTriggerAutomationTool_Execute_MissingAutomationID(t *testing.T) {
	tool := NewTriggerAutomationTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "automation_id")
}

func TestTriggerAutomationTool_Execute_MissingLeadID(t *testing.T) {
	tool := NewTriggerAutomationTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"automation_id": "auto-99",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "lead_id")
}

func TestTriggerAutomationTool_Execute_RepositoryError(t *testing.T) {
	trigger := &fakeAutomationTrigger{err: errors.New("timeout na automacao")}
	tool := NewTriggerAutomationTool(trigger)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"automation_id": "auto-99",
		"lead_id":       "lead-42",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "timeout na automacao")
}

// ─── SwitchSpecialistTool tests ───────────────────────────────────────────────

func TestSwitchSpecialistTool_Definition(t *testing.T) {
	tool := NewSwitchSpecialistTool(nil)
	def := tool.Definition()

	assert.Equal(t, "switch_specialist", def.Name)
	assert.Equal(t, domain.ToolCategoryAutomation, def.Category)
	assert.Contains(t, def.Parameters, "specialist_id")
	assert.True(t, def.Parameters["specialist_id"].Required)
	assert.Contains(t, def.Parameters, "conversation_id")
	assert.True(t, def.Parameters["conversation_id"].Required)
}

func TestSwitchSpecialistTool_Execute_HappyPath(t *testing.T) {
	switcher := &fakeSpecialistSwitcher{}
	tool := NewSwitchSpecialistTool(switcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"specialist_id":   "spec-7",
		"conversation_id": "conv-123",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "spec-7")
	assert.Contains(t, result.Content, "conv-123")
	assert.Contains(t, result.Content, "sucesso")
}

func TestSwitchSpecialistTool_Execute_MissingSpecialistID(t *testing.T) {
	tool := NewSwitchSpecialistTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"conversation_id": "conv-123",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "specialist_id")
}

func TestSwitchSpecialistTool_Execute_MissingConversationID(t *testing.T) {
	tool := NewSwitchSpecialistTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"specialist_id": "spec-7",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "conversation_id")
}

func TestSwitchSpecialistTool_Execute_RepositoryError(t *testing.T) {
	switcher := &fakeSpecialistSwitcher{err: errors.New("specialist nao encontrado")}
	tool := NewSwitchSpecialistTool(switcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"specialist_id":   "spec-7",
		"conversation_id": "conv-123",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "specialist nao encontrado")
}
