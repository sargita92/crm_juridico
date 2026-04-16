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

type fakeLeadMover struct {
	err error
}

func (f *fakeLeadMover) MoveToColumn(_ context.Context, _, _, _ string) error {
	return f.err
}

type fakeNoteCreator struct {
	capturedCreatedBy string
	err               error
}

func (f *fakeNoteCreator) CreateNote(_ context.Context, _, _, _, createdBy string) error {
	f.capturedCreatedBy = createdBy
	return f.err
}

type fakeScoreUpdater struct {
	capturedScore int
	err           error
}

func (f *fakeScoreUpdater) UpdateScore(_ context.Context, _, _ string, score int) error {
	f.capturedScore = score
	return f.err
}

// ─── MoveLeadTool tests ───────────────────────────────────────────────────────

func TestMoveLeadTool_Definition(t *testing.T) {
	tool := NewMoveLeadTool(nil)
	def := tool.Definition()

	assert.Equal(t, "move_lead", def.Name)
	assert.Equal(t, domain.ToolCategoryCRMAction, def.Category)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.True(t, def.Parameters["lead_id"].Required)
	assert.Contains(t, def.Parameters, "column_id")
	assert.True(t, def.Parameters["column_id"].Required)
}

func TestMoveLeadTool_Execute_HappyPath(t *testing.T) {
	mover := &fakeLeadMover{}
	tool := NewMoveLeadTool(mover)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id":   "lead-42",
		"column_id": "col-99",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "lead-42")
	assert.Contains(t, result.Content, "col-99")
	assert.Contains(t, result.Content, "sucesso")
}

func TestMoveLeadTool_Execute_MissingLeadID(t *testing.T) {
	tool := NewMoveLeadTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"column_id": "col-99",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "lead_id")
}

func TestMoveLeadTool_Execute_MissingColumnID(t *testing.T) {
	tool := NewMoveLeadTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "column_id")
}

func TestMoveLeadTool_Execute_RepositoryError(t *testing.T) {
	mover := &fakeLeadMover{err: errors.New("db timeout")}
	tool := NewMoveLeadTool(mover)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id":   "lead-42",
		"column_id": "col-99",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "db timeout")
}

// ─── CreateLeadNoteTool tests ─────────────────────────────────────────────────

func TestCreateLeadNoteTool_Definition(t *testing.T) {
	tool := NewCreateLeadNoteTool(nil)
	def := tool.Definition()

	assert.Equal(t, "create_note", def.Name)
	assert.Equal(t, domain.ToolCategoryCRMAction, def.Category)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.True(t, def.Parameters["lead_id"].Required)
	assert.Contains(t, def.Parameters, "content")
	assert.True(t, def.Parameters["content"].Required)
}

func TestCreateLeadNoteTool_Execute_HappyPath(t *testing.T) {
	creator := &fakeNoteCreator{}
	tool := NewCreateLeadNoteTool(creator)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"content": "Cliente interessado no produto premium",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "lead-42")
	assert.Contains(t, result.Content, "sucesso")
}

func TestCreateLeadNoteTool_Execute_CreatedByIsAlwaysAISpecialist(t *testing.T) {
	creator := &fakeNoteCreator{}
	tool := NewCreateLeadNoteTool(creator)

	_, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"content": "nota qualquer",
	})
	require.NoError(t, err)
	assert.Equal(t, "ai_specialist", creator.capturedCreatedBy)
}

func TestCreateLeadNoteTool_Execute_MissingLeadID(t *testing.T) {
	tool := NewCreateLeadNoteTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"content": "alguma nota",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "lead_id")
}

func TestCreateLeadNoteTool_Execute_MissingContent(t *testing.T) {
	tool := NewCreateLeadNoteTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "content")
}

func TestCreateLeadNoteTool_Execute_RepositoryError(t *testing.T) {
	creator := &fakeNoteCreator{err: errors.New("connection refused")}
	tool := NewCreateLeadNoteTool(creator)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"content": "nota qualquer",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "connection refused")
}

// ─── UpdateLeadScoreTool tests ────────────────────────────────────────────────

func TestUpdateLeadScoreTool_Definition(t *testing.T) {
	tool := NewUpdateLeadScoreTool(nil)
	def := tool.Definition()

	assert.Equal(t, "update_score", def.Name)
	assert.Equal(t, domain.ToolCategoryCRMAction, def.Category)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.True(t, def.Parameters["lead_id"].Required)
	assert.Contains(t, def.Parameters, "score")
	assert.True(t, def.Parameters["score"].Required)
}

func TestUpdateLeadScoreTool_Execute_HappyPath_Float64(t *testing.T) {
	updater := &fakeScoreUpdater{}
	tool := NewUpdateLeadScoreTool(updater)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"score":   float64(85),
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "lead-42")
	assert.Contains(t, result.Content, "85")
	assert.Equal(t, 85, updater.capturedScore)
}

func TestUpdateLeadScoreTool_Execute_HappyPath_Int(t *testing.T) {
	updater := &fakeScoreUpdater{}
	tool := NewUpdateLeadScoreTool(updater)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"score":   int(72),
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, 72, updater.capturedScore)
}

func TestUpdateLeadScoreTool_Execute_MissingLeadID(t *testing.T) {
	tool := NewUpdateLeadScoreTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"score": float64(50),
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "lead_id")
}

func TestUpdateLeadScoreTool_Execute_MissingScore(t *testing.T) {
	tool := NewUpdateLeadScoreTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "score")
}

func TestUpdateLeadScoreTool_Execute_InvalidScoreType(t *testing.T) {
	tool := NewUpdateLeadScoreTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"score":   "noventa", // string instead of number
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "score")
}

func TestUpdateLeadScoreTool_Execute_RepositoryError(t *testing.T) {
	updater := &fakeScoreUpdater{err: errors.New("update failed")}
	tool := NewUpdateLeadScoreTool(updater)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id": "lead-42",
		"score":   float64(60),
	})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "update failed")
}
