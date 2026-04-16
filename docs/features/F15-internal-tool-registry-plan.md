# Internal Tool Registry — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give AI specialists the ability to execute tools (data queries, CRM actions, automations) during WhatsApp conversations via a provider-agnostic internal tool registry.

**Architecture:** Tool definitions live in the AI domain layer. A ToolRegistry maps names to implementations. A ToolResolver filters tools per specialist (via DB association) and per step (via forced/restricted fields). The ConversationEngine runs a tool-calling loop: send request with tools to provider, receive tool calls, execute, inject results, repeat until text response. Each provider converts ToolDefinition to its native format internally.

**Tech Stack:** Go, Gin, GORM, MySQL, golang-migrate, testify, Prometheus, Zap, OpenTelemetry

**Design Spec:** `docs/features/F15-internal-tool-registry-design.md`

---

## File Structure

### New Files

| File | Responsibility |
|---|---|
| `internal/ai/domain/tool.go` | ToolDefinition, ToolCall, ToolResult, ToolCategory, ParameterDef, Tool interface |
| `internal/ai/domain/tool_test.go` | Domain type validation tests |
| `internal/ai/application/tool_registry.go` | ToolRegistry — register, get, list tools |
| `internal/ai/application/tool_registry_test.go` | Registry tests |
| `internal/ai/application/tool_resolver.go` | ToolResolver — filter by specialist + step constraints |
| `internal/ai/application/tool_resolver_test.go` | Resolver tests |
| `internal/specialist/domain/specialist_tool.go` | SpecialistTool join entity |
| `internal/specialist/infrastructure/gorm_specialist_tool_repository.go` | GORM repository for specialist-tool associations |
| `internal/specialist/infrastructure/gorm_specialist_tool_repository_test.go` | Repository tests |
| `internal/ai/infrastructure/tools/search_leads.go` | SearchLeadsTool |
| `internal/ai/infrastructure/tools/get_lead_detail.go` | GetLeadDetailTool |
| `internal/ai/infrastructure/tools/get_conversation_history.go` | GetConversationHistoryTool |
| `internal/ai/infrastructure/tools/list_products.go` | ListProductsTool |
| `internal/ai/infrastructure/tools/get_pipeline.go` | GetPipelineTool |
| `internal/ai/infrastructure/tools/move_lead.go` | MoveLeadTool |
| `internal/ai/infrastructure/tools/create_note.go` | CreateLeadNoteTool |
| `internal/ai/infrastructure/tools/update_score.go` | UpdateLeadScoreTool |
| `internal/ai/infrastructure/tools/trigger_automation.go` | TriggerAutomationTool |
| `internal/ai/infrastructure/tools/switch_specialist.go` | SwitchSpecialistTool |
| `internal/ai/infrastructure/tools/tools_test.go` | Tests for all concrete tools |
| `internal/specialist/interfaces/http/tool_handler.go` | Admin UI handler for tool associations |
| `web/templates/specialist/tools.html` | HTMX template for tool checkboxes |
| `migrations/000049_create_specialist_tools.up.sql` | Migration: specialist_tools table |
| `migrations/000049_create_specialist_tools.down.sql` | Rollback |
| `migrations/000050_add_step_tool_fields.up.sql` | Migration: forced_tools, restricted_tools on steps |
| `migrations/000050_add_step_tool_fields.down.sql` | Rollback |

### Modified Files

| File | Change |
|---|---|
| `internal/ai/domain/provider.go:44-80` | Add Tools, ToolResults to AIRequest; ToolCalls to AIResponse |
| `internal/ai/domain/provider_test.go` | Update tests for new fields |
| `internal/ai/infrastructure/openai_provider.go:47-181` | Add buildTools, parseToolCalls, tool message handling |
| `internal/ai/application/conversation_engine.go:33-45,48-73,126-133` | Add toolRegistry field, executeToolLoop method, replace direct GenerateResponse call |
| `internal/ai/application/context_builder.go:50-58,61-77,80-162` | Add ToolResolver dep, resolve tools in Build() |
| `internal/ai/application/metrics.go` | Add 4 new tool metrics |
| `internal/specialist/domain/step.go:17-28` | Add ForcedTools, RestrictedTools fields |
| `internal/specialist/infrastructure/gorm_step_repository.go` | Handle JSON serialization of tool fields |
| `internal/shared/config/config.go:21-30` | Add tool limit fields to AIConfigEnv |
| `internal/ai/module.go:29-48,77-226` | Extend ModuleDeps, wire tools in NewModule |
| `cmd/api/main.go:84-168` | Pass new deps to AI module |
| `internal/specialist/module.go` | Expose SpecialistToolRepository |
| `internal/specialist/interfaces/http/handler.go` | Register tool routes |

---

## Task 1: Domain Types — ToolDefinition, ToolCall, ToolResult, Tool Interface

**Files:**
- Create: `internal/ai/domain/tool.go`
- Test: `internal/ai/domain/tool_test.go`

- [ ] **Step 1: Write the failing test for ToolDefinition validation**

```go
// internal/ai/domain/tool_test.go
package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolDefinition_Valid(t *testing.T) {
	td, err := NewToolDefinition("search_leads", "Search leads by query", ToolCategoryDataQuery, map[string]ParameterDef{
		"query": {Type: "string", Description: "Search term", Required: true},
	})
	require.NoError(t, err)
	assert.Equal(t, "search_leads", td.Name)
	assert.Equal(t, ToolCategoryDataQuery, td.Category)
	assert.Len(t, td.Parameters, 1)
}

func TestNewToolDefinition_EmptyName(t *testing.T) {
	_, err := NewToolDefinition("", "desc", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolNameRequired)
}

func TestNewToolDefinition_EmptyDescription(t *testing.T) {
	_, err := NewToolDefinition("name", "", ToolCategoryDataQuery, nil)
	assert.ErrorIs(t, err, ErrToolDescriptionRequired)
}

func TestNewToolDefinition_InvalidCategory(t *testing.T) {
	_, err := NewToolDefinition("name", "desc", ToolCategory("invalid"), nil)
	assert.ErrorIs(t, err, ErrToolCategoryInvalid)
}

func TestNewToolResult_Valid(t *testing.T) {
	r := NewToolResult("call-1", "result text", false)
	assert.Equal(t, "call-1", r.ToolCallID)
	assert.Equal(t, "result text", r.Content)
	assert.False(t, r.IsError)
}

func TestNewToolResult_Error(t *testing.T) {
	r := NewToolResult("call-1", "something failed", true)
	assert.True(t, r.IsError)
}

func TestNewToolResult_Truncate(t *testing.T) {
	long := make([]byte, 5000)
	for i := range long {
		long[i] = 'a'
	}
	r := NewToolResultWithLimit("call-1", string(long), false, 4000)
	assert.Len(t, r.Content, 4000)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/domain/ -run TestNewToolDefinition -v`
Expected: FAIL — `tool.go` does not exist

- [ ] **Step 3: Write the implementation**

```go
// internal/ai/domain/tool.go
package domain

import (
	"context"
	"errors"
)

var (
	ErrToolNameRequired        = errors.New("tool name is required")
	ErrToolDescriptionRequired = errors.New("tool description is required")
	ErrToolCategoryInvalid     = errors.New("tool category is invalid")
	ErrToolNotFound            = errors.New("tool not found")
)

// ToolCategory groups tools by their purpose.
type ToolCategory string

const (
	ToolCategoryDataQuery  ToolCategory = "data_query"
	ToolCategoryCRMAction  ToolCategory = "crm_action"
	ToolCategoryAutomation ToolCategory = "automation"
)

func isValidToolCategory(c ToolCategory) bool {
	switch c {
	case ToolCategoryDataQuery, ToolCategoryCRMAction, ToolCategoryAutomation:
		return true
	}
	return false
}

// ParameterDef describes a single parameter for a tool.
type ParameterDef struct {
	Type        string   // "string", "number", "boolean"
	Description string
	Required    bool
	Enum        []string
}

// ToolDefinition describes a tool that an AI specialist can invoke.
type ToolDefinition struct {
	Name        string
	Description string
	Category    ToolCategory
	Parameters  map[string]ParameterDef
}

// NewToolDefinition creates a ToolDefinition with validation.
func NewToolDefinition(name, description string, category ToolCategory, params map[string]ParameterDef) (ToolDefinition, error) {
	if name == "" {
		return ToolDefinition{}, ErrToolNameRequired
	}
	if description == "" {
		return ToolDefinition{}, ErrToolDescriptionRequired
	}
	if !isValidToolCategory(category) {
		return ToolDefinition{}, ErrToolCategoryInvalid
	}
	if params == nil {
		params = make(map[string]ParameterDef)
	}
	return ToolDefinition{
		Name:        name,
		Description: description,
		Category:    category,
		Parameters:  params,
	}, nil
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID        string
	ToolName  string
	Arguments map[string]interface{}
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// NewToolResult creates a ToolResult.
func NewToolResult(toolCallID, content string, isError bool) ToolResult {
	return ToolResult{ToolCallID: toolCallID, Content: content, IsError: isError}
}

// NewToolResultWithLimit creates a ToolResult, truncating content if it exceeds maxLen.
func NewToolResultWithLimit(toolCallID, content string, isError bool, maxLen int) ToolResult {
	if maxLen > 0 && len(content) > maxLen {
		content = content[:maxLen]
	}
	return ToolResult{ToolCallID: toolCallID, Content: content, IsError: isError}
}

// Tool is the interface that all concrete tools must implement.
type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*ToolResult, error)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ai/domain/ -run "TestNewTool" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/domain/tool.go internal/ai/domain/tool_test.go
git commit -m "feat(F15): add tool domain types — ToolDefinition, ToolCall, ToolResult, Tool interface"
```

---

## Task 2: Extend AIRequest and AIResponse

**Files:**
- Modify: `internal/ai/domain/provider.go:44-80`
- Modify: `internal/ai/domain/provider_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Add to internal/ai/domain/provider_test.go

func TestAIRequest_WithTools(t *testing.T) {
	td, _ := NewToolDefinition("test_tool", "A test tool", ToolCategoryDataQuery, nil)
	msg, _ := NewAIMessage(RoleUser, "hello")
	req, err := NewAIRequest("system", []AIMessage{msg}, "openai", "gpt-4", 0.7, 1024)
	require.NoError(t, err)

	req.Tools = []ToolDefinition{td}
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "test_tool", req.Tools[0].Name)
}

func TestAIRequest_WithToolResults(t *testing.T) {
	msg, _ := NewAIMessage(RoleUser, "hello")
	req, err := NewAIRequest("system", []AIMessage{msg}, "openai", "gpt-4", 0.7, 1024)
	require.NoError(t, err)

	req.ToolResults = []ToolResult{NewToolResult("call-1", "data", false)}
	assert.Len(t, req.ToolResults, 1)
}

func TestAIResponse_WithToolCalls(t *testing.T) {
	resp := &AIResponse{
		Content:      "",
		FinishReason: "tool_calls",
		ToolCalls: []ToolCall{
			{ID: "call-1", ToolName: "search_leads", Arguments: map[string]interface{}{"query": "test"}},
		},
	}
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "search_leads", resp.ToolCalls[0].ToolName)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/domain/ -run "TestAIRequest_WithTools|TestAIRequest_WithToolResults|TestAIResponse_WithToolCalls" -v`
Expected: FAIL — fields don't exist on AIRequest/AIResponse

- [ ] **Step 3: Modify AIRequest and AIResponse**

In `internal/ai/domain/provider.go`, update the `AIRequest` struct (line 44) to:

```go
// AIRequest represents a request sent to an AI provider.
type AIRequest struct {
	SystemPrompt string
	Messages     []AIMessage
	Provider     string
	Model        string
	Temperature  float64
	MaxTokens    int
	Tools        []ToolDefinition
	ToolResults  []ToolResult
}
```

Update the `AIResponse` struct (line 75) to:

```go
// AIResponse represents the response from an AI provider.
type AIResponse struct {
	Content          string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	ToolCalls        []ToolCall
}
```

- [ ] **Step 4: Run all domain tests to verify they pass**

Run: `go test ./internal/ai/domain/ -v`
Expected: PASS (existing tests should still pass, new fields are zero-value)

- [ ] **Step 5: Commit**

```bash
git add internal/ai/domain/provider.go internal/ai/domain/provider_test.go
git commit -m "feat(F15): extend AIRequest with Tools/ToolResults, AIResponse with ToolCalls"
```

---

## Task 3: Config Extension — Tool Limits

**Files:**
- Modify: `internal/shared/config/config.go:21-30`

- [ ] **Step 1: Add tool config fields to AIConfigEnv**

In `internal/shared/config/config.go`, extend the `AIConfigEnv` struct:

```go
type AIConfigEnv struct {
	OpenAIAPIKey            string
	DefaultProvider         string
	DefaultModel            string
	DefaultMaxTokens        int
	DefaultTemperature      float64
	DefaultDebounce         int
	PlaygroundEnabled       bool
	ResetCommandEnabled     bool
	ToolLoopMaxIterations   int
	ToolCallMaxPerIteration int
	ToolExecutionTimeout    int
	ToolResultMaxLength     int
}
```

- [ ] **Step 2: Add defaults in the config loading function**

In the same file, where Viper defaults are set, add:

```go
viper.SetDefault("ai.tool_loop_max_iterations", 5)
viper.SetDefault("ai.tool_call_max_per_iteration", 10)
viper.SetDefault("ai.tool_execution_timeout", 10)
viper.SetDefault("ai.tool_result_max_length", 4000)
```

And in the config loading section, read the values:

```go
cfg.AI.ToolLoopMaxIterations = viper.GetInt("ai.tool_loop_max_iterations")
cfg.AI.ToolCallMaxPerIteration = viper.GetInt("ai.tool_call_max_per_iteration")
cfg.AI.ToolExecutionTimeout = viper.GetInt("ai.tool_execution_timeout")
cfg.AI.ToolResultMaxLength = viper.GetInt("ai.tool_result_max_length")
```

Also support env vars: `AI_TOOL_LOOP_MAX_ITERATIONS`, `AI_TOOL_CALL_MAX_PER_ITERATION`, `AI_TOOL_EXECUTION_TIMEOUT_SECONDS`, `AI_TOOL_RESULT_MAX_LENGTH`.

- [ ] **Step 3: Verify build compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/shared/config/config.go
git commit -m "feat(F15): add tool loop config limits to AIConfigEnv"
```

---

## Task 4: Migrations — specialist_tools table and step tool fields

**Files:**
- Create: `migrations/000049_create_specialist_tools.up.sql`
- Create: `migrations/000049_create_specialist_tools.down.sql`
- Create: `migrations/000050_add_step_tool_fields.up.sql`
- Create: `migrations/000050_add_step_tool_fields.down.sql`

- [ ] **Step 1: Create specialist_tools migration (up)**

```sql
-- migrations/000049_create_specialist_tools.up.sql
CREATE TABLE specialist_tools (
    id            CHAR(36) NOT NULL PRIMARY KEY,
    specialist_id CHAR(36) NOT NULL,
    tool_name     VARCHAR(100) NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_specialist_tools_specialist FOREIGN KEY (specialist_id) REFERENCES specialists(id) ON DELETE CASCADE,
    UNIQUE KEY idx_specialist_tool (specialist_id, tool_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

- [ ] **Step 2: Create specialist_tools migration (down)**

```sql
-- migrations/000049_create_specialist_tools.down.sql
DROP TABLE IF EXISTS specialist_tools;
```

- [ ] **Step 3: Create step tool fields migration (up)**

```sql
-- migrations/000050_add_step_tool_fields.up.sql
ALTER TABLE steps
    ADD COLUMN forced_tools JSON NULL AFTER target_column_id,
    ADD COLUMN restricted_tools JSON NULL AFTER forced_tools;
```

- [ ] **Step 4: Create step tool fields migration (down)**

```sql
-- migrations/000050_add_step_tool_fields.down.sql
ALTER TABLE steps
    DROP COLUMN restricted_tools,
    DROP COLUMN forced_tools;
```

- [ ] **Step 5: Run migrations to verify they apply**

Run: `go run cmd/api/main.go migrate up` (or however migrations are run in this project)
Expected: Migrations 49 and 50 applied successfully

- [ ] **Step 6: Commit**

```bash
git add migrations/000049_create_specialist_tools.up.sql migrations/000049_create_specialist_tools.down.sql migrations/000050_add_step_tool_fields.up.sql migrations/000050_add_step_tool_fields.down.sql
git commit -m "feat(F15): add migrations for specialist_tools table and step tool fields"
```

---

## Task 5: SpecialistTool Entity and Repository

**Files:**
- Create: `internal/specialist/domain/specialist_tool.go`
- Create: `internal/specialist/infrastructure/gorm_specialist_tool_repository.go`
- Create: `internal/specialist/infrastructure/gorm_specialist_tool_repository_test.go`

- [ ] **Step 1: Write the failing test for the repository**

```go
// internal/specialist/infrastructure/gorm_specialist_tool_repository_test.go
package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGormSpecialistToolRepository_AssociateAndFind(t *testing.T) {
	db := setupTestDB(t) // use the existing test DB helper in this package
	repo := NewGormSpecialistToolRepository(db)
	ctx := context.Background()

	specialistID := uuid.New().String()
	createTestSpecialist(t, db, specialistID) // helper to insert a specialist row

	err := repo.Associate(ctx, specialistID, "search_leads")
	require.NoError(t, err)

	err = repo.Associate(ctx, specialistID, "move_lead")
	require.NoError(t, err)

	tools, err := repo.FindToolNamesBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Len(t, tools, 2)
	assert.Contains(t, tools, "search_leads")
	assert.Contains(t, tools, "move_lead")
}

func TestGormSpecialistToolRepository_Dissociate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormSpecialistToolRepository(db)
	ctx := context.Background()

	specialistID := uuid.New().String()
	createTestSpecialist(t, db, specialistID)

	_ = repo.Associate(ctx, specialistID, "search_leads")
	err := repo.Dissociate(ctx, specialistID, "search_leads")
	require.NoError(t, err)

	tools, _ := repo.FindToolNamesBySpecialistID(ctx, specialistID)
	assert.Empty(t, tools)
}

func TestGormSpecialistToolRepository_DuplicateAssociation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewGormSpecialistToolRepository(db)
	ctx := context.Background()

	specialistID := uuid.New().String()
	createTestSpecialist(t, db, specialistID)

	_ = repo.Associate(ctx, specialistID, "search_leads")
	err := repo.Associate(ctx, specialistID, "search_leads")
	assert.ErrorIs(t, err, domain.ErrToolAlreadyAssociated)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/specialist/infrastructure/ -run TestGormSpecialistToolRepository -v`
Expected: FAIL — files don't exist

- [ ] **Step 3: Create the SpecialistTool domain entity**

```go
// internal/specialist/domain/specialist_tool.go
package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrToolAlreadyAssociated = errors.New("tool is already associated with this specialist")
	ErrToolNotAssociated     = errors.New("tool is not associated with this specialist")
)

// SpecialistTool represents the association between a specialist and a tool.
type SpecialistTool struct {
	ID           string
	SpecialistID string
	ToolName     string
	CreatedAt    time.Time
}

// SpecialistToolRepository defines persistence for specialist-tool associations.
type SpecialistToolRepository interface {
	Associate(ctx context.Context, specialistID, toolName string) error
	Dissociate(ctx context.Context, specialistID, toolName string) error
	FindToolNamesBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
	FindAll(ctx context.Context, specialistID string) ([]SpecialistTool, error)
}
```

- [ ] **Step 4: Create the GORM repository implementation**

Follow the exact pattern from `internal/mcp/infrastructure/gorm_specialist_mcp_repository.go`:

```go
// internal/specialist/infrastructure/gorm_specialist_tool_repository.go
package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"gorm.io/gorm"
)

type specialistToolModel struct {
	ID           string    `gorm:"column:id;primaryKey"`
	SpecialistID string    `gorm:"column:specialist_id"`
	ToolName     string    `gorm:"column:tool_name"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (specialistToolModel) TableName() string { return "specialist_tools" }

type GormSpecialistToolRepository struct {
	db *gorm.DB
}

func NewGormSpecialistToolRepository(db *gorm.DB) *GormSpecialistToolRepository {
	return &GormSpecialistToolRepository{db: db}
}

func (r *GormSpecialistToolRepository) Associate(ctx context.Context, specialistID, toolName string) error {
	model := specialistToolModel{
		ID:           uuid.New().String(),
		SpecialistID: specialistID,
		ToolName:     toolName,
		CreatedAt:    time.Now(),
	}
	err := r.db.WithContext(ctx).Create(&model).Error
	if err != nil && strings.Contains(err.Error(), "Duplicate") {
		return domain.ErrToolAlreadyAssociated
	}
	return err
}

func (r *GormSpecialistToolRepository) Dissociate(ctx context.Context, specialistID, toolName string) error {
	result := r.db.WithContext(ctx).
		Where("specialist_id = ? AND tool_name = ?", specialistID, toolName).
		Delete(&specialistToolModel{})
	if result.RowsAffected == 0 {
		return domain.ErrToolNotAssociated
	}
	return result.Error
}

func (r *GormSpecialistToolRepository) FindToolNamesBySpecialistID(ctx context.Context, specialistID string) ([]string, error) {
	var models []specialistToolModel
	err := r.db.WithContext(ctx).
		Where("specialist_id = ?", specialistID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.ToolName
	}
	return names, nil
}

func (r *GormSpecialistToolRepository) FindAll(ctx context.Context, specialistID string) ([]domain.SpecialistTool, error) {
	var models []specialistToolModel
	err := r.db.WithContext(ctx).
		Where("specialist_id = ?", specialistID).
		Find(&models).Error
	if err != nil {
		return nil, err
	}
	result := make([]domain.SpecialistTool, len(models))
	for i, m := range models {
		result[i] = domain.SpecialistTool{
			ID:           m.ID,
			SpecialistID: m.SpecialistID,
			ToolName:     m.ToolName,
			CreatedAt:    m.CreatedAt,
		}
	}
	return result, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/specialist/infrastructure/ -run TestGormSpecialistToolRepository -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/specialist/domain/specialist_tool.go internal/specialist/infrastructure/gorm_specialist_tool_repository.go internal/specialist/infrastructure/gorm_specialist_tool_repository_test.go
git commit -m "feat(F15): add SpecialistTool entity and GORM repository"
```

---

## Task 6: Step Domain Extension — ForcedTools, RestrictedTools

**Files:**
- Modify: `internal/specialist/domain/step.go:17-28`
- Modify: `internal/specialist/infrastructure/gorm_step_repository.go` (GORM model)

- [ ] **Step 1: Write the failing test**

```go
// Add to existing step test file or create if needed

func TestStep_WithToolFields(t *testing.T) {
	s, err := NewStep(uuid.New().String(), uuid.New().String(), 0, "Collect name", StepDataTypeFreeText, true, 10, "")
	require.NoError(t, err)

	s.ForcedTools = []string{"search_leads"}
	s.RestrictedTools = []string{"move_lead"}

	assert.Equal(t, []string{"search_leads"}, s.ForcedTools)
	assert.Equal(t, []string{"move_lead"}, s.RestrictedTools)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/specialist/domain/ -run TestStep_WithToolFields -v`
Expected: FAIL — ForcedTools/RestrictedTools don't exist on Step

- [ ] **Step 3: Add fields to Step struct**

In `internal/specialist/domain/step.go`, update the Step struct (line 17):

```go
type Step struct {
	ID              string
	SpecialistID    string
	OrderIndex      int
	Text            string
	DataType        StepDataType
	Required        bool
	Score           int
	TargetColumnID  string
	ForcedTools     []string
	RestrictedTools []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

- [ ] **Step 4: Update GORM step model for JSON serialization**

In the step repository's GORM model, add the JSON columns. The model should use `datatypes.JSON` or a custom scanner. Since the project uses raw JSON strings (like McpServer.Config), use the same pattern:

```go
// In the step GORM model struct, add:
ForcedTools     string `gorm:"column:forced_tools;type:json"`
RestrictedTools string `gorm:"column:restricted_tools;type:json"`
```

In the toDomain/toModel conversion functions, marshal/unmarshal `[]string` to/from JSON string:

```go
import "encoding/json"

// toModel: convert []string → JSON string
func marshalToolList(tools []string) string {
	if len(tools) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(tools)
	return string(b)
}

// toDomain: convert JSON string → []string
func unmarshalToolList(raw string) []string {
	if raw == "" || raw == "null" {
		return nil
	}
	var tools []string
	_ = json.Unmarshal([]byte(raw), &tools)
	return tools
}
```

- [ ] **Step 5: Run all step tests**

Run: `go test ./internal/specialist/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/specialist/domain/step.go internal/specialist/infrastructure/gorm_step_repository.go
git commit -m "feat(F15): add ForcedTools and RestrictedTools fields to Step entity"
```

---

## Task 7: ToolRegistry

**Files:**
- Create: `internal/ai/application/tool_registry.go`
- Create: `internal/ai/application/tool_registry_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/ai/application/tool_registry_test.go
package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTool implements domain.Tool for testing.
type fakeTool struct {
	def domain.ToolDefinition
}

func (f *fakeTool) Definition() domain.ToolDefinition { return f.def }
func (f *fakeTool) Execute(_ context.Context, _ string, _ map[string]interface{}) (*domain.ToolResult, error) {
	r := domain.NewToolResult("", "ok", false)
	return &r, nil
}

func newFakeTool(name string, cat domain.ToolCategory) *fakeTool {
	def, _ := domain.NewToolDefinition(name, "desc for "+name, cat, nil)
	return &fakeTool{def: def}
}

func TestToolRegistry_RegisterAndGet(t *testing.T) {
	reg := NewToolRegistry()
	tool := newFakeTool("search_leads", domain.ToolCategoryDataQuery)

	reg.Register(tool)

	got, err := reg.Get("search_leads")
	require.NoError(t, err)
	assert.Equal(t, "search_leads", got.Definition().Name)
}

func TestToolRegistry_GetNotFound(t *testing.T) {
	reg := NewToolRegistry()
	_, err := reg.Get("nonexistent")
	assert.ErrorIs(t, err, domain.ErrToolNotFound)
}

func TestToolRegistry_All(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(newFakeTool("a", domain.ToolCategoryDataQuery))
	reg.Register(newFakeTool("b", domain.ToolCategoryCRMAction))

	all := reg.All()
	assert.Len(t, all, 2)
}

func TestToolRegistry_Definitions(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(newFakeTool("a", domain.ToolCategoryDataQuery))
	reg.Register(newFakeTool("b", domain.ToolCategoryCRMAction))

	defs := reg.Definitions()
	assert.Len(t, defs, 2)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/application/ -run TestToolRegistry -v`
Expected: FAIL — tool_registry.go doesn't exist

- [ ] **Step 3: Write the implementation**

```go
// internal/ai/application/tool_registry.go
package application

import (
	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// ToolRegistry holds all registered tools by name.
type ToolRegistry struct {
	tools map[string]domain.Tool
}

// NewToolRegistry creates an empty ToolRegistry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]domain.Tool),
	}
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool domain.Tool) {
	r.tools[tool.Definition().Name] = tool
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (domain.Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, domain.ErrToolNotFound
	}
	return t, nil
}

// All returns all registered tools.
func (r *ToolRegistry) All() []domain.Tool {
	result := make([]domain.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Definitions returns ToolDefinitions for all registered tools.
func (r *ToolRegistry) Definitions() []domain.ToolDefinition {
	result := make([]domain.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t.Definition())
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ai/application/ -run TestToolRegistry -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/application/tool_registry.go internal/ai/application/tool_registry_test.go
git commit -m "feat(F15): add ToolRegistry for tool registration and lookup"
```

---

## Task 8: ToolResolver — Filter by Specialist and Step

**Files:**
- Create: `internal/ai/application/tool_resolver.go`
- Create: `internal/ai/application/tool_resolver_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/ai/application/tool_resolver_test.go
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

	resolver := NewToolResolver(reg, finder)
	return resolver, reg
}

func TestToolResolver_ResolveForSpecialist(t *testing.T) {
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

func TestToolResolver_ResolveForSpecialist_NoAssociations(t *testing.T) {
	resolver, _ := setupResolver()
	ctx := context.Background()

	tools, err := resolver.ResolveForSpecialist(ctx, "spec-unknown")
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestToolResolver_ApplyStepConstraints_ForcedTools(t *testing.T) {
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

func TestToolResolver_ApplyStepConstraints_RestrictedTools(t *testing.T) {
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

func TestToolResolver_ApplyStepConstraints_NilStep(t *testing.T) {
	resolver, _ := setupResolver()

	tools := []domain.Tool{
		newFakeTool("search_leads", domain.ToolCategoryDataQuery),
	}

	filtered := resolver.ApplyStepConstraints(tools, nil)
	assert.Len(t, filtered, 1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/application/ -run TestToolResolver -v`
Expected: FAIL — tool_resolver.go doesn't exist

- [ ] **Step 3: Write the implementation**

```go
// internal/ai/application/tool_resolver.go
package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// SpecialistToolFinder retrieves tool names associated with a specialist.
type SpecialistToolFinder interface {
	FindToolNamesBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
}

// ToolResolver filters tools for a specialist and applies step constraints.
type ToolResolver struct {
	registry *ToolRegistry
	finder   SpecialistToolFinder
}

// NewToolResolver creates a ToolResolver.
func NewToolResolver(registry *ToolRegistry, finder SpecialistToolFinder) *ToolResolver {
	return &ToolResolver{registry: registry, finder: finder}
}

// ResolveForSpecialist returns the tools available for a given specialist.
func (r *ToolResolver) ResolveForSpecialist(ctx context.Context, specialistID string) ([]domain.Tool, error) {
	names, err := r.finder.FindToolNamesBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	var tools []domain.Tool
	for _, name := range names {
		tool, tErr := r.registry.Get(name)
		if tErr != nil {
			continue // tool removed from registry but still in DB — skip
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// ApplyStepConstraints filters tools based on step's ForcedTools and RestrictedTools.
func (r *ToolResolver) ApplyStepConstraints(tools []domain.Tool, step *specDomain.Step) []domain.Tool {
	if step == nil {
		return tools
	}

	// If ForcedTools is set, only allow those tools (intersect with available).
	if len(step.ForcedTools) > 0 {
		forced := make(map[string]bool, len(step.ForcedTools))
		for _, name := range step.ForcedTools {
			forced[name] = true
		}
		var filtered []domain.Tool
		for _, t := range tools {
			if forced[t.Definition().Name] {
				filtered = append(filtered, t)
			}
		}
		tools = filtered
	}

	// If RestrictedTools is set, remove those tools.
	if len(step.RestrictedTools) > 0 {
		restricted := make(map[string]bool, len(step.RestrictedTools))
		for _, name := range step.RestrictedTools {
			restricted[name] = true
		}
		var filtered []domain.Tool
		for _, t := range tools {
			if !restricted[t.Definition().Name] {
				filtered = append(filtered, t)
			}
		}
		tools = filtered
	}

	return tools
}

// ResolveDefinitions returns ToolDefinitions (not Tools) for use in AIRequest.
func (r *ToolResolver) ResolveDefinitions(ctx context.Context, specialistID string, step *specDomain.Step) ([]domain.ToolDefinition, error) {
	tools, err := r.ResolveForSpecialist(ctx, specialistID)
	if err != nil {
		return nil, err
	}
	tools = r.ApplyStepConstraints(tools, step)

	defs := make([]domain.ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = t.Definition()
	}
	return defs, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ai/application/ -run TestToolResolver -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/application/tool_resolver.go internal/ai/application/tool_resolver_test.go
git commit -m "feat(F15): add ToolResolver for per-specialist and per-step tool filtering"
```

---

## Task 9: Prometheus Metrics for Tools

**Files:**
- Modify: `internal/ai/application/metrics.go`

- [ ] **Step 1: Add 4 new metrics**

Add after the existing metrics in `internal/ai/application/metrics.go`:

```go
var (
	aiToolCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "ai",
		Name:      "tool_calls_total",
		Help:      "Total number of tool calls executed",
	}, []string{"tenant_id", "specialist_id", "tool_name", "status"})

	aiToolCallDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "crm",
		Subsystem: "ai",
		Name:      "tool_call_duration_seconds",
		Help:      "Duration of tool call execution",
		Buckets:   prometheus.DefBuckets,
	}, []string{"tenant_id", "tool_name"})

	aiToolLoopIterations = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "crm",
		Subsystem: "ai",
		Name:      "tool_loop_iterations",
		Help:      "Number of iterations in the tool calling loop",
		Buckets:   []float64{1, 2, 3, 4, 5},
	}, []string{"tenant_id", "specialist_id"})

	aiToolResultTruncatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "crm",
		Subsystem: "ai",
		Name:      "tool_result_truncated_total",
		Help:      "Total number of tool results that were truncated",
	}, []string{"tenant_id", "tool_name"})
)
```

Register them in `init()` alongside existing metrics.

- [ ] **Step 2: Verify build compiles**

Run: `go build ./internal/ai/...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/ai/application/metrics.go
git commit -m "feat(F15): add Prometheus metrics for tool calls, duration, loop iterations"
```

---

## Task 10: OpenAI Provider — Tool Support

**Files:**
- Modify: `internal/ai/infrastructure/openai_provider.go`
- Create: `internal/ai/infrastructure/openai_provider_tools_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/ai/infrastructure/openai_provider_tools_test.go
package infrastructure

import (
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOpenAITools(t *testing.T) {
	p := &OpenAIProvider{}
	td, _ := domain.NewToolDefinition("search_leads", "Search leads", domain.ToolCategoryDataQuery, map[string]domain.ParameterDef{
		"query":  {Type: "string", Description: "Search term", Required: true},
		"status": {Type: "string", Description: "Lead status", Required: false, Enum: []string{"open", "won", "lost"}},
	})

	tools := p.buildOpenAITools([]domain.ToolDefinition{td})

	require.Len(t, tools, 1)
	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "search_leads", tools[0].Function.Name)
	assert.Equal(t, "Search leads", tools[0].Function.Description)

	params := tools[0].Function.Parameters
	assert.Equal(t, "object", params["type"])
	props := params["properties"].(map[string]interface{})
	assert.Contains(t, props, "query")
	assert.Contains(t, props, "status")

	queryProp := props["query"].(map[string]interface{})
	assert.Equal(t, "string", queryProp["type"])

	statusProp := props["status"].(map[string]interface{})
	assert.Equal(t, []string{"open", "won", "lost"}, statusProp["enum"])

	required := params["required"].([]string)
	assert.Contains(t, required, "query")
	assert.NotContains(t, required, "status")
}

func TestParseToolCalls(t *testing.T) {
	p := &OpenAIProvider{}
	choices := []openAIChoice{
		{
			Message: openAIMessage{
				Role:      "assistant",
				Content:   "",
				ToolCalls: []openAIToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: openAIFunctionCall{
							Name:      "search_leads",
							Arguments: `{"query":"test","status":"open"}`,
						},
					},
				},
			},
			FinishReason: "tool_calls",
		},
	}

	calls := p.parseToolCalls(choices)
	require.Len(t, calls, 1)
	assert.Equal(t, "call_123", calls[0].ID)
	assert.Equal(t, "search_leads", calls[0].ToolName)
	assert.Equal(t, "test", calls[0].Arguments["query"])
	assert.Equal(t, "open", calls[0].Arguments["status"])
}

func TestBuildToolResultMessages(t *testing.T) {
	p := &OpenAIProvider{}
	results := []domain.ToolResult{
		{ToolCallID: "call_123", Content: "found 3 leads", IsError: false},
	}

	msgs := p.buildToolResultMessages(results)
	require.Len(t, msgs, 1)
	assert.Equal(t, "tool", msgs[0].Role)
	assert.Equal(t, "call_123", msgs[0].ToolCallID)
	assert.Equal(t, "found 3 leads", msgs[0].Content)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/infrastructure/ -run "TestBuildOpenAITools|TestParseToolCalls|TestBuildToolResultMessages" -v`
Expected: FAIL — methods don't exist

- [ ] **Step 3: Update JSON structs to support tool calls**

In `internal/ai/infrastructure/openai_provider.go`, replace the internal JSON structs (lines 147-180) with:

```go
// — internal JSON structs —

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type openAIError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
```

- [ ] **Step 4: Add buildOpenAITools method**

```go
// buildOpenAITools converts domain ToolDefinitions to OpenAI function calling format.
func (p *OpenAIProvider) buildOpenAITools(tools []domain.ToolDefinition) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openAITool, len(tools))
	for i, td := range tools {
		properties := make(map[string]interface{})
		var required []string

		for name, param := range td.Parameters {
			prop := map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if len(param.Enum) > 0 {
				prop["enum"] = param.Enum
			}
			properties[name] = prop
			if param.Required {
				required = append(required, name)
			}
		}

		params := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			params["required"] = required
		}

		result[i] = openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        td.Name,
				Description: td.Description,
				Parameters:  params,
			},
		}
	}
	return result
}
```

- [ ] **Step 5: Add parseToolCalls method**

```go
// parseToolCalls extracts domain ToolCalls from OpenAI response choices.
func (p *OpenAIProvider) parseToolCalls(choices []openAIChoice) []domain.ToolCall {
	if len(choices) == 0 {
		return nil
	}
	msg := choices[0].Message
	if len(msg.ToolCalls) == 0 {
		return nil
	}

	calls := make([]domain.ToolCall, len(msg.ToolCalls))
	for i, tc := range msg.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if args == nil {
			args = make(map[string]interface{})
		}
		calls[i] = domain.ToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: args,
		}
	}
	return calls
}
```

- [ ] **Step 6: Add buildToolResultMessages method**

```go
// buildToolResultMessages converts domain ToolResults to OpenAI tool messages.
func (p *OpenAIProvider) buildToolResultMessages(results []domain.ToolResult) []openAIMessage {
	msgs := make([]openAIMessage, len(results))
	for i, r := range results {
		msgs[i] = openAIMessage{
			Role:       "tool",
			ToolCallID: r.ToolCallID,
			Content:    r.Content,
		}
	}
	return msgs
}
```

- [ ] **Step 7: Update GenerateResponse to use tools**

In the `GenerateResponse` method (line 47), update the request body construction:

```go
func (p *OpenAIProvider) GenerateResponse(ctx context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	messages := p.buildMessages(req)

	// Append tool result messages if present.
	messages = append(messages, p.buildToolResultMessages(req.ToolResults)...)

	body := openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Tools:       p.buildOpenAITools(req.Tools),
	}

	// ... rest of the method stays the same until response parsing ...
```

Update the response parsing section (after `choice := apiResp.Choices[0]`) to extract tool calls:

```go
	toolCalls := p.parseToolCalls(apiResp.Choices)

	return &domain.AIResponse{
		Content:          choice.Message.Content,
		FinishReason:     choice.FinishReason,
		PromptTokens:     apiResp.Usage.PromptTokens,
		CompletionTokens: apiResp.Usage.CompletionTokens,
		ToolCalls:        toolCalls,
	}, nil
```

- [ ] **Step 8: Update buildMessages to include assistant tool call messages**

When the provider is continuing a tool loop, the assistant's previous message (with tool calls) must be included. Update `buildMessages`:

```go
func (p *OpenAIProvider) buildMessages(req *domain.AIRequest) []openAIMessage {
	var messages []openAIMessage
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    string(domain.RoleSystem),
			Content: req.SystemPrompt,
		})
	}
	for _, m := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	return messages
}
```

- [ ] **Step 9: Run tests to verify they pass**

Run: `go test ./internal/ai/infrastructure/ -run "TestBuildOpenAITools|TestParseToolCalls|TestBuildToolResultMessages" -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add internal/ai/infrastructure/openai_provider.go internal/ai/infrastructure/openai_provider_tools_test.go
git commit -m "feat(F15): add OpenAI function calling support — buildTools, parseToolCalls, tool messages"
```

---

## Task 11: Tool Calling Loop in ConversationEngine

**Files:**
- Modify: `internal/ai/application/conversation_engine.go:33-45,48-73,126-133`
- Create: `internal/ai/application/tool_loop_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/ai/application/tool_loop_test.go
package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeProvider simulates an AI provider that returns tool calls then text.
type fakeToolProvider struct {
	callCount int
	responses []*domain.AIResponse
}

func (f *fakeToolProvider) Name() string { return "fake" }
func (f *fakeToolProvider) GenerateResponse(_ context.Context, _ *domain.AIRequest) (*domain.AIResponse, error) {
	if f.callCount >= len(f.responses) {
		return nil, fmt.Errorf("unexpected call %d", f.callCount)
	}
	resp := f.responses[f.callCount]
	f.callCount++
	return resp, nil
}

func TestExecuteToolLoop_NoTools(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{Content: "Hello!", FinishReason: "stop"},
		},
	}
	registry := NewToolRegistry()
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          log,
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "hi"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "Hello!", resp.Content)
}

func TestExecuteToolLoop_WithToolCall(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls: []domain.ToolCall{
					{ID: "call-1", ToolName: "fake_tool", Arguments: map[string]interface{}{"q": "test"}},
				},
			},
			{Content: "Found results!", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	registry.Register(newFakeTool("fake_tool", domain.ToolCategoryDataQuery))
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          log,
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "search"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "Found results!", resp.Content)
	assert.Equal(t, 2, provider.callCount)
}

func TestExecuteToolLoop_MaxIterations(t *testing.T) {
	// Provider always returns tool calls — should hit max iterations.
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c1", ToolName: "fake_tool"}}},
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c2", ToolName: "fake_tool"}}},
			{FinishReason: "tool_calls", ToolCalls: []domain.ToolCall{{ID: "c3", ToolName: "fake_tool"}}},
		},
	}

	registry := NewToolRegistry()
	registry.Register(newFakeTool("fake_tool", domain.ToolCategoryDataQuery))
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          log,
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "loop"}}}
	_, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 3, 4000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max iterations")
}

func TestExecuteToolLoop_ToolNotFound(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{
				FinishReason: "tool_calls",
				ToolCalls:    []domain.ToolCall{{ID: "c1", ToolName: "nonexistent"}},
			},
			{Content: "OK I couldn't find that tool", FinishReason: "stop"},
		},
	}

	registry := NewToolRegistry()
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry: registry,
		log:          log,
	}

	req := &domain.AIRequest{Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "test"}}}
	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "OK I couldn't find that tool", resp.Content)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/application/ -run TestExecuteToolLoop -v`
Expected: FAIL — executeToolLoop doesn't exist, toolRegistry field doesn't exist

- [ ] **Step 3: Add toolRegistry field to ConversationEngine**

In `internal/ai/application/conversation_engine.go`, add to the struct (after line 44):

```go
type ConversationEngine struct {
	providerRegistry    *domain.ProviderRegistry
	configResolver      ConfigResolver
	stateRepo           domain.ConversationStateRepository
	contextBuilder      *ContextBuilder
	stepEvaluator       *StepEvaluator
	guardrailChecker    *GuardrailChecker
	messageSender       MessageSender
	leadUpdater         LeadUpdater
	resetUC             *ResetConversationUseCase
	resetCommandEnabled bool
	toolRegistry        *ToolRegistry
	toolResultMaxLength int
	toolLoopMaxIter     int
	log                 *zap.Logger
}
```

Update `NewConversationEngine` to accept the new params:

```go
func NewConversationEngine(
	providerRegistry *domain.ProviderRegistry,
	configResolver ConfigResolver,
	stateRepo domain.ConversationStateRepository,
	contextBuilder *ContextBuilder,
	stepEvaluator *StepEvaluator,
	guardrailChecker *GuardrailChecker,
	messageSender MessageSender,
	leadUpdater LeadUpdater,
	resetUC *ResetConversationUseCase,
	resetCommandEnabled bool,
	toolRegistry *ToolRegistry,
	toolResultMaxLength int,
	toolLoopMaxIter int,
	log *zap.Logger,
) *ConversationEngine {
	return &ConversationEngine{
		providerRegistry:    providerRegistry,
		configResolver:      configResolver,
		stateRepo:           stateRepo,
		contextBuilder:      contextBuilder,
		stepEvaluator:       stepEvaluator,
		guardrailChecker:    guardrailChecker,
		messageSender:       messageSender,
		leadUpdater:         leadUpdater,
		resetUC:             resetUC,
		resetCommandEnabled: resetCommandEnabled,
		toolRegistry:        toolRegistry,
		toolResultMaxLength: toolResultMaxLength,
		toolLoopMaxIter:     toolLoopMaxIter,
		log:                 log,
	}
}
```

- [ ] **Step 4: Implement executeToolLoop**

Add the method to `conversation_engine.go`:

```go
// executeToolLoop runs the tool calling loop: send request → get tool calls → execute → repeat.
func (e *ConversationEngine) executeToolLoop(
	ctx context.Context,
	provider domain.AIProvider,
	req *domain.AIRequest,
	tenantID, specialistID string,
	maxIterations int,
	resultMaxLength int,
) (*domain.AIResponse, error) {
	for i := 0; i < maxIterations; i++ {
		resp, err := provider.GenerateResponse(ctx, req)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			aiToolLoopIterations.WithLabelValues(tenantID, specialistID).Observe(float64(i + 1))
			return resp, nil
		}

		var results []domain.ToolResult
		for _, call := range resp.ToolCalls {
			tool, tErr := e.toolRegistry.Get(call.ToolName)
			if tErr != nil {
				aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "not_found").Inc()
				results = append(results, domain.NewToolResult(call.ID, "tool not found: "+call.ToolName, true))
				continue
			}

			start := time.Now()
			result, execErr := tool.Execute(ctx, tenantID, call.Arguments)
			elapsed := time.Since(start)
			aiToolCallDuration.WithLabelValues(tenantID, call.ToolName).Observe(elapsed.Seconds())

			if execErr != nil {
				aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "error").Inc()
				e.log.Warn("tool_call_failed",
					zap.String("tenant_id", tenantID),
					zap.String("tool_name", call.ToolName),
					zap.Error(execErr),
				)
				results = append(results, domain.NewToolResult(call.ID, "error: "+execErr.Error(), true))
				continue
			}

			// Truncate if needed.
			if resultMaxLength > 0 && len(result.Content) > resultMaxLength {
				aiToolResultTruncatedTotal.WithLabelValues(tenantID, call.ToolName).Inc()
				result.Content = result.Content[:resultMaxLength]
			}

			aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "success").Inc()
			e.log.Info("tool_call_executed",
				zap.String("tenant_id", tenantID),
				zap.String("specialist_id", specialistID),
				zap.String("tool_name", call.ToolName),
				zap.Duration("duration", elapsed),
			)
			results = append(results, *result)
		}

		req.ToolResults = results
	}

	e.log.Warn("tool_loop_max_iterations",
		zap.String("tenant_id", tenantID),
		zap.Int("max_iterations", maxIterations),
	)
	return nil, fmt.Errorf("tool loop exceeded max iterations (%d)", maxIterations)
}
```

- [ ] **Step 5: Replace direct GenerateResponse call in HandleMessages**

In `HandleMessages`, replace lines 131-133:

```go
	// Before:
	// start := time.Now()
	// resp, err := provider.GenerateResponse(ctx, req)
	// elapsed := time.Since(start).Seconds()

	// After:
	start := time.Now()
	resp, err := e.executeToolLoop(ctx, provider, req, tenantID, specialistID, e.toolLoopMaxIter, e.toolResultMaxLength)
	elapsed := time.Since(start).Seconds()
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/ai/application/ -run TestExecuteToolLoop -v`
Expected: PASS

- [ ] **Step 7: Run all existing tests to check for regressions**

Run: `go test ./internal/ai/... -v`
Expected: PASS (update any existing tests that call NewConversationEngine with the new params)

- [ ] **Step 8: Commit**

```bash
git add internal/ai/application/conversation_engine.go internal/ai/application/tool_loop_test.go
git commit -m "feat(F15): add executeToolLoop to ConversationEngine with metrics and error handling"
```

---

## Task 12: ContextBuilder — Inject Tools into AIRequest

**Files:**
- Modify: `internal/ai/application/context_builder.go:50-58,61-77,80-162`

- [ ] **Step 1: Write the failing test**

```go
// Add to existing context_builder_test.go or create new file

func TestContextBuilder_BuildWithTools(t *testing.T) {
	// Setup fakes for all existing deps + toolResolver
	// The key assertion: req.Tools is populated when toolResolver returns definitions.

	fakeToolResolver := &fakeToolResolverForBuilder{
		defs: []domain.ToolDefinition{
			{Name: "search_leads", Description: "Search", Category: domain.ToolCategoryDataQuery},
		},
	}

	builder := &ContextBuilder{
		SpecialistFinder:     /* existing fake */,
		StepFinder:           /* existing fake */,
		GuardrailFinder:      /* existing fake */,
		DocumentFetcher:      /* existing fake */,
		ProductInfoFinder:    /* existing fake */,
		MessageHistoryFinder: /* existing fake */,
		ToolResolver:         fakeToolResolver,
	}

	state := &domain.ConversationState{SpecialistID: "spec-1", ConversationID: "conv-1"}
	req, err := builder.Build(context.Background(), state, "", 20)
	require.NoError(t, err)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "search_leads", req.Tools[0].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/application/ -run TestContextBuilder_BuildWithTools -v`
Expected: FAIL — ToolResolver field doesn't exist on ContextBuilder

- [ ] **Step 3: Add ToolResolver to ContextBuilder**

In `internal/ai/application/context_builder.go`, update the struct (line 51):

```go
type ContextBuilder struct {
	SpecialistFinder     SpecialistFinder
	StepFinder           StepFinder
	GuardrailFinder      GuardrailFinder
	DocumentFetcher      DocumentFetcher
	ProductInfoFinder    ProductInfoFinder
	MessageHistoryFinder MessageHistoryFinder
	ToolResolver         *ToolResolver
}
```

Update `NewContextBuilder` to accept `ToolResolver`:

```go
func NewContextBuilder(
	specialistFinder SpecialistFinder,
	stepFinder StepFinder,
	guardrailFinder GuardrailFinder,
	documentFetcher DocumentFetcher,
	productInfoFinder ProductInfoFinder,
	messageHistoryFinder MessageHistoryFinder,
	toolResolver *ToolResolver,
) *ContextBuilder {
	return &ContextBuilder{
		SpecialistFinder:     specialistFinder,
		StepFinder:           stepFinder,
		GuardrailFinder:      guardrailFinder,
		DocumentFetcher:      documentFetcher,
		ProductInfoFinder:    productInfoFinder,
		MessageHistoryFinder: messageHistoryFinder,
		ToolResolver:         toolResolver,
	}
}
```

- [ ] **Step 4: Add tool resolution to Build method**

After step 8 (fallback message, line 156) and before the return, add:

```go
	// 9. Resolve tools for this specialist and current step.
	var toolDefs []domain.ToolDefinition
	if b.ToolResolver != nil {
		var currentStep *specDomain.Step
		if len(steps) > 0 && state.CurrentStepIndex < len(steps) {
			currentStep = &steps[state.CurrentStepIndex]
		}
		toolDefs, _ = b.ToolResolver.ResolveDefinitions(ctx, state.SpecialistID, currentStep)
	}

	return &domain.AIRequest{
		SystemPrompt: systemPrompt,
		Messages:     messages,
		Tools:        toolDefs,
	}, nil
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/ai/application/ -v`
Expected: PASS (fix any existing tests that call NewContextBuilder with the new param — pass `nil` for ToolResolver where not needed)

- [ ] **Step 6: Commit**

```bash
git add internal/ai/application/context_builder.go
git commit -m "feat(F15): inject tool definitions into AIRequest via ContextBuilder"
```

---

## Task 13: Concrete Tools — Data Query Category

**Files:**
- Create: `internal/ai/infrastructure/tools/search_leads.go`
- Create: `internal/ai/infrastructure/tools/get_lead_detail.go`
- Create: `internal/ai/infrastructure/tools/get_conversation_history.go`
- Create: `internal/ai/infrastructure/tools/list_products.go`
- Create: `internal/ai/infrastructure/tools/get_pipeline.go`
- Create: `internal/ai/infrastructure/tools/data_query_test.go`

- [ ] **Step 1: Write the failing test for SearchLeadsTool**

```go
// internal/ai/infrastructure/tools/data_query_test.go
package tools

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLeadSearcher struct {
	leads []funnelDomain.Lead
}

func (f *fakeLeadSearcher) SearchByTenant(ctx context.Context, tenantID, query string) ([]funnelDomain.Lead, error) {
	return f.leads, nil
}

func TestSearchLeadsTool_Definition(t *testing.T) {
	tool := NewSearchLeadsTool(nil)
	def := tool.Definition()
	assert.Equal(t, "search_leads", def.Name)
	assert.Equal(t, domain.ToolCategoryDataQuery, def.Category)
	assert.Contains(t, def.Parameters, "query")
}

func TestSearchLeadsTool_Execute(t *testing.T) {
	searcher := &fakeLeadSearcher{
		leads: []funnelDomain.Lead{
			{ID: "l1", Score: 80, Status: funnelDomain.LeadStatusOpen},
		},
	}
	tool := NewSearchLeadsTool(searcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{"query": "test"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content, "l1")
}

func TestSearchLeadsTool_MissingQuery(t *testing.T) {
	tool := NewSearchLeadsTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content, "query")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/infrastructure/tools/ -run TestSearchLeadsTool -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement SearchLeadsTool**

```go
// internal/ai/infrastructure/tools/search_leads.go
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// LeadSearcher searches leads by tenant.
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
	def, _ := domain.NewToolDefinition("search_leads", "Busca leads por nome, telefone ou status", domain.ToolCategoryDataQuery, map[string]domain.ParameterDef{
		"query": {Type: "string", Description: "Termo de busca (nome ou telefone)", Required: true},
		"status": {Type: "string", Description: "Filtrar por status", Required: false, Enum: []string{"open", "won", "lost"}},
	})
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
```

- [ ] **Step 4: Implement remaining data query tools**

Implement each following the same pattern:

**GetLeadDetailTool** (`get_lead_detail.go`):
- Interface: `LeadDetailFetcher` with `FindByID(ctx, tenantID, leadID) (*Lead, error)`
- Params: `lead_id` (required)
- Returns: full lead info (ID, score, status, column, contact info, notes)

**GetConversationHistoryTool** (`get_conversation_history.go`):
- Interface: `ConversationHistoryFetcher` with `FindMessages(ctx, conversationID, limit) ([]Message, error)`
- Params: `conversation_id` (required), `limit` (optional, default 20)
- Returns: formatted message list with timestamps

**ListProductsTool** (`list_products.go`):
- Interface: `ProductLister` with `FindByTenantID(ctx, tenantID) ([]Product, error)`
- Params: none
- Returns: product list with name, description, keywords

**GetPipelineTool** (`get_pipeline.go`):
- Interface: `PipelineFetcher` with `FindFunnelWithColumns(ctx, tenantID, funnelID) (*Funnel, []Column, error)`
- Params: `funnel_id` (optional)
- Returns: funnel columns with lead counts per stage

- [ ] **Step 5: Run all data query tests**

Run: `go test ./internal/ai/infrastructure/tools/ -run "TestSearchLeads|TestGetLeadDetail|TestGetConversationHistory|TestListProducts|TestGetPipeline" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/infrastructure/tools/
git commit -m "feat(F15): implement data query tools — SearchLeads, GetLeadDetail, GetHistory, ListProducts, GetPipeline"
```

---

## Task 14: Concrete Tools — CRM Action Category

**Files:**
- Create: `internal/ai/infrastructure/tools/move_lead.go`
- Create: `internal/ai/infrastructure/tools/create_note.go`
- Create: `internal/ai/infrastructure/tools/update_score.go`
- Create: `internal/ai/infrastructure/tools/crm_action_test.go`

- [ ] **Step 1: Write the failing test for MoveLeadTool**

```go
// internal/ai/infrastructure/tools/crm_action_test.go
package tools

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLeadMover struct {
	called   bool
	leadID   string
	columnID string
}

func (f *fakeLeadMover) MoveToColumn(ctx context.Context, tenantID, leadID, columnID string) error {
	f.called = true
	f.leadID = leadID
	f.columnID = columnID
	return nil
}

func TestMoveLeadTool_Definition(t *testing.T) {
	tool := NewMoveLeadTool(nil)
	def := tool.Definition()
	assert.Equal(t, "move_lead", def.Name)
	assert.Equal(t, domain.ToolCategoryCRMAction, def.Category)
	assert.Contains(t, def.Parameters, "lead_id")
	assert.Contains(t, def.Parameters, "column_id")
}

func TestMoveLeadTool_Execute(t *testing.T) {
	mover := &fakeLeadMover{}
	tool := NewMoveLeadTool(mover)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"lead_id":   "lead-123",
		"column_id": "col-456",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.True(t, mover.called)
	assert.Equal(t, "lead-123", mover.leadID)
}

func TestMoveLeadTool_MissingParams(t *testing.T) {
	tool := NewMoveLeadTool(nil)
	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{})
	require.NoError(t, err)
	assert.True(t, result.IsError)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/infrastructure/tools/ -run TestMoveLeadTool -v`
Expected: FAIL

- [ ] **Step 3: Implement MoveLeadTool**

```go
// internal/ai/infrastructure/tools/move_lead.go
package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type LeadMover interface {
	MoveToColumn(ctx context.Context, tenantID, leadID, columnID string) error
}

type MoveLeadTool struct {
	mover LeadMover
}

func NewMoveLeadTool(mover LeadMover) *MoveLeadTool {
	return &MoveLeadTool{mover: mover}
}

func (t *MoveLeadTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition("move_lead", "Move um lead para outra coluna do kanban", domain.ToolCategoryCRMAction, map[string]domain.ParameterDef{
		"lead_id":   {Type: "string", Description: "ID do lead", Required: true},
		"column_id": {Type: "string", Description: "ID da coluna destino", Required: true},
	})
	return def
}

func (t *MoveLeadTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	leadID, _ := args["lead_id"].(string)
	columnID, _ := args["column_id"].(string)
	if leadID == "" || columnID == "" {
		r := domain.NewToolResult("", "parametros obrigatorios: 'lead_id' e 'column_id'", true)
		return &r, nil
	}

	if err := t.mover.MoveToColumn(ctx, tenantID, leadID, columnID); err != nil {
		r := domain.NewToolResult("", "erro ao mover lead: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Lead %s movido para coluna %s com sucesso", leadID, columnID), false)
	return &r, nil
}
```

- [ ] **Step 4: Implement CreateLeadNoteTool and UpdateLeadScoreTool**

Follow the same pattern:

**CreateLeadNoteTool** (`create_note.go`):
- Interface: `NoteCreator` with `CreateNote(ctx, tenantID, leadID, content, createdBy string) error`
- Params: `lead_id` (required), `content` (required)
- `createdBy` is set to `"ai_specialist"` internally

**UpdateLeadScoreTool** (`update_score.go`):
- Interface: `ScoreUpdater` with `UpdateScore(ctx, tenantID, leadID string, score int) error`
- Params: `lead_id` (required), `score` (required, number)
- Parse `score` from `float64` (JSON numbers are float64 in Go)

- [ ] **Step 5: Run CRM action tests**

Run: `go test ./internal/ai/infrastructure/tools/ -run "TestMoveLeadTool|TestCreateLeadNoteTool|TestUpdateLeadScoreTool" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/infrastructure/tools/move_lead.go internal/ai/infrastructure/tools/create_note.go internal/ai/infrastructure/tools/update_score.go internal/ai/infrastructure/tools/crm_action_test.go
git commit -m "feat(F15): implement CRM action tools — MoveLead, CreateLeadNote, UpdateLeadScore"
```

---

## Task 15: Concrete Tools — Automation Category

**Files:**
- Create: `internal/ai/infrastructure/tools/trigger_automation.go`
- Create: `internal/ai/infrastructure/tools/switch_specialist.go`
- Create: `internal/ai/infrastructure/tools/automation_test.go`

- [ ] **Step 1: Write the failing test for TriggerAutomationTool**

```go
// internal/ai/infrastructure/tools/automation_test.go
package tools

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAutomationTrigger struct {
	triggered bool
}

func (f *fakeAutomationTrigger) TriggerManually(ctx context.Context, tenantID, automationID, leadID string) (string, error) {
	f.triggered = true
	return "automation executed", nil
}

func TestTriggerAutomationTool_Execute(t *testing.T) {
	trigger := &fakeAutomationTrigger{}
	tool := NewTriggerAutomationTool(trigger)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"automation_id": "auto-1",
		"lead_id":       "lead-1",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.True(t, trigger.triggered)
}

type fakeSpecialistSwitcher struct {
	switched bool
}

func (f *fakeSpecialistSwitcher) SwitchSpecialist(ctx context.Context, conversationID, specialistID string) error {
	f.switched = true
	return nil
}

func TestSwitchSpecialistTool_Execute(t *testing.T) {
	switcher := &fakeSpecialistSwitcher{}
	tool := NewSwitchSpecialistTool(switcher)

	result, err := tool.Execute(context.Background(), "tenant-1", map[string]interface{}{
		"specialist_id": "spec-2",
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.True(t, switcher.switched)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/infrastructure/tools/ -run "TestTriggerAutomationTool|TestSwitchSpecialistTool" -v`
Expected: FAIL

- [ ] **Step 3: Implement TriggerAutomationTool**

```go
// internal/ai/infrastructure/tools/trigger_automation.go
package tools

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type AutomationTrigger interface {
	TriggerManually(ctx context.Context, tenantID, automationID, leadID string) (string, error)
}

type TriggerAutomationTool struct {
	trigger AutomationTrigger
}

func NewTriggerAutomationTool(trigger AutomationTrigger) *TriggerAutomationTool {
	return &TriggerAutomationTool{trigger: trigger}
}

func (t *TriggerAutomationTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition("trigger_automation", "Dispara uma automacao manualmente para um lead", domain.ToolCategoryAutomation, map[string]domain.ParameterDef{
		"automation_id": {Type: "string", Description: "ID da automacao", Required: true},
		"lead_id":       {Type: "string", Description: "ID do lead", Required: true},
	})
	return def
}

func (t *TriggerAutomationTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	automationID, _ := args["automation_id"].(string)
	leadID, _ := args["lead_id"].(string)
	if automationID == "" || leadID == "" {
		r := domain.NewToolResult("", "parametros obrigatorios: 'automation_id' e 'lead_id'", true)
		return &r, nil
	}

	output, err := t.trigger.TriggerManually(ctx, tenantID, automationID, leadID)
	if err != nil {
		r := domain.NewToolResult("", "erro ao disparar automacao: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", output, false)
	return &r, nil
}
```

- [ ] **Step 4: Implement SwitchSpecialistTool**

```go
// internal/ai/infrastructure/tools/switch_specialist.go
package tools

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

type SpecialistSwitcher interface {
	SwitchSpecialist(ctx context.Context, conversationID, specialistID string) error
}

type SwitchSpecialistTool struct {
	switcher SpecialistSwitcher
}

func NewSwitchSpecialistTool(switcher SpecialistSwitcher) *SwitchSpecialistTool {
	return &SwitchSpecialistTool{switcher: switcher}
}

func (t *SwitchSpecialistTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition("switch_specialist", "Troca o especialista no meio da conversa", domain.ToolCategoryAutomation, map[string]domain.ParameterDef{
		"specialist_id": {Type: "string", Description: "ID do novo especialista", Required: true},
	})
	return def
}

func (t *SwitchSpecialistTool) Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*domain.ToolResult, error) {
	specialistID, _ := args["specialist_id"].(string)
	if specialistID == "" {
		r := domain.NewToolResult("", "parametro obrigatorio: 'specialist_id'", true)
		return &r, nil
	}

	// Note: conversationID must be injected via context or a wrapper.
	// For now, the SwitchSpecialist interface handles the lookup.
	if err := t.switcher.SwitchSpecialist(ctx, "", specialistID); err != nil {
		r := domain.NewToolResult("", "erro ao trocar especialista: "+err.Error(), true)
		return &r, nil
	}

	r := domain.NewToolResult("", fmt.Sprintf("Especialista trocado para %s com sucesso", specialistID), false)
	return &r, nil
}
```

- [ ] **Step 5: Run automation tests**

Run: `go test ./internal/ai/infrastructure/tools/ -run "TestTriggerAutomation|TestSwitchSpecialist" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/infrastructure/tools/trigger_automation.go internal/ai/infrastructure/tools/switch_specialist.go internal/ai/infrastructure/tools/automation_test.go
git commit -m "feat(F15): implement automation tools — TriggerAutomation, SwitchSpecialist"
```

---

## Task 16: Module Wiring

**Files:**
- Modify: `internal/ai/module.go:29-48,77-226`
- Modify: `cmd/api/main.go:84-168`
- Modify: `internal/specialist/module.go`

- [ ] **Step 1: Expose SpecialistToolRepository from specialist module**

In `internal/specialist/module.go`, add the repo field and accessor:

```go
// Add to Module struct:
specialistToolRepo domain.SpecialistToolRepository

// In NewModule, create the repo:
specialistToolRepo := infrastructure.NewGormSpecialistToolRepository(db)

// Add accessor method:
func (m *Module) SpecialistToolRepo() domain.SpecialistToolRepository {
	return m.specialistToolRepo
}
```

- [ ] **Step 2: Extend AI ModuleDeps**

In `internal/ai/module.go`, add to `ModuleDeps`:

```go
type ModuleDeps struct {
	// ... existing deps ...
	SpecialistToolFinder application.SpecialistToolFinder
}
```

- [ ] **Step 3: Wire tools in AI NewModule**

In the `NewModule` function, after creating existing components:

```go
	// Create tool registry and register all tools.
	toolRegistry := application.NewToolRegistry()

	// Create tool adapters (wrapping existing use cases/repos from deps).
	// Each tool gets the specific interface it needs via adapter.
	searchLeadsTool := tools.NewSearchLeadsTool(/* adapter wrapping LeadRepository */)
	getLeadDetailTool := tools.NewGetLeadDetailTool(/* adapter */)
	getHistoryTool := tools.NewGetConversationHistoryTool(/* adapter */)
	listProductsTool := tools.NewListProductsTool(/* adapter */)
	getPipelineTool := tools.NewGetPipelineTool(/* adapter */)
	moveLeadTool := tools.NewMoveLeadTool(/* adapter */)
	createNoteTool := tools.NewCreateLeadNoteTool(/* adapter */)
	updateScoreTool := tools.NewUpdateLeadScoreTool(/* adapter */)
	triggerAutomationTool := tools.NewTriggerAutomationTool(/* adapter */)
	switchSpecialistTool := tools.NewSwitchSpecialistTool(/* adapter */)

	toolRegistry.Register(searchLeadsTool)
	toolRegistry.Register(getLeadDetailTool)
	toolRegistry.Register(getHistoryTool)
	toolRegistry.Register(listProductsTool)
	toolRegistry.Register(getPipelineTool)
	toolRegistry.Register(moveLeadTool)
	toolRegistry.Register(createNoteTool)
	toolRegistry.Register(updateScoreTool)
	toolRegistry.Register(triggerAutomationTool)
	toolRegistry.Register(switchSpecialistTool)

	// Create tool resolver.
	toolResolver := application.NewToolResolver(toolRegistry, deps.SpecialistToolFinder)

	// Pass toolResolver to ContextBuilder.
	contextBuilder := application.NewContextBuilder(
		// ... existing deps ...,
		toolResolver,
	)

	// Pass toolRegistry to ConversationEngine.
	engine := application.NewConversationEngine(
		// ... existing deps ...,
		toolRegistry,
		cfg.ToolResultMaxLength,
		cfg.ToolLoopMaxIterations,
		log,
	)
```

- [ ] **Step 4: Update cmd/api/main.go to pass new deps**

In `cmd/api/main.go`, where the AI module is created (around line 130):

```go
	aiMod := ai.NewModule(db, cfg.AI, log, ai.ModuleDeps{
		// ... existing deps ...
		SpecialistToolFinder: specialistMod.SpecialistToolRepo(),
	})
```

- [ ] **Step 5: Create tool adapters**

Create adapter files in `internal/ai/infrastructure/` that wrap existing repositories/use cases to satisfy the tool interfaces. For example:

```go
// internal/ai/infrastructure/lead_search_adapter.go
package infrastructure

import (
	"context"

	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type LeadSearchAdapter struct {
	repo funnelDomain.LeadRepository
}

func NewLeadSearchAdapter(repo funnelDomain.LeadRepository) *LeadSearchAdapter {
	return &LeadSearchAdapter{repo: repo}
}

func (a *LeadSearchAdapter) SearchByTenant(ctx context.Context, tenantID, query string) ([]funnelDomain.Lead, error) {
	// Note: LeadRepository.FindByFunnelID requires a funnelID.
	// This adapter needs a method that searches across all funnels in a tenant.
	// If this doesn't exist yet, add FindByTenantID(tenantID, filter) to LeadRepository.
	list, err := a.repo.FindByTenantID(ctx, tenantID, funnelDomain.LeadFilter{Search: query})
	if err != nil {
		return nil, err
	}
	return list.Leads, nil
}
```

Create one adapter per tool interface. Each adapter wraps an existing repository or use case, translating the tool's interface to the existing API. Pattern:

**LeadDetailAdapter** — wraps `LeadRepository.FindByID` + `LeadNoteRepository.FindByLeadID`
**ConversationHistoryAdapter** — wraps `MessageRepository.FindByConversationID` (already exists as `MessageHistoryFinder`)
**ProductListAdapter** — wraps `ProductRepository.FindByTenantID`
**PipelineAdapter** — wraps `FunnelRepository.FindByTenantID` + `ColumnRepository.FindByFunnelID` + `LeadRepository.CountByColumnID`
**LeadMoveAdapter** — wraps `MoveLeadUseCase.Execute`
**NoteCreatorAdapter** — wraps `LeadNoteRepository.Create`
**ScoreUpdaterAdapter** — wraps `LeadRepository.FindByID` + `Lead.UpdateScore` + `LeadRepository.Update`
**AutomationTriggerAdapter** — wraps `AutomationEngine.ExecuteManually`
**SpecialistSwitcherAdapter** — wraps `ConversationStateRepository.FindByConversationID` + update SpecialistID

If `LeadRepository` doesn't have `FindByTenantID`, add it as part of this task (new method on the interface + GORM implementation).

- [ ] **Step 6: Verify build compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 7: Run all tests**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/ai/module.go internal/ai/infrastructure/lead_search_adapter.go internal/ai/infrastructure/*_adapter.go internal/specialist/module.go cmd/api/main.go
git commit -m "feat(F15): wire tool registry, resolver, and all tools into AI module"
```

---

## Task 17: Admin UI — Specialist Tool Association

**Files:**
- Create: `internal/specialist/interfaces/http/tool_handler.go`
- Create: `web/templates/specialist/tools.html`
- Modify: `internal/specialist/interfaces/http/handler.go` (register routes)
- Modify: `internal/specialist/module.go` (create handler)

- [ ] **Step 1: Create the tool handler**

```go
// internal/specialist/interfaces/http/tool_handler.go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ToolHandler struct {
	toolRepo     domain.SpecialistToolRepository
	toolRegistry *application.ToolRegistry
}

func NewToolHandler(toolRepo domain.SpecialistToolRepository, toolRegistry *application.ToolRegistry) *ToolHandler {
	return &ToolHandler{toolRepo: toolRepo, toolRegistry: toolRegistry}
}

func (h *ToolHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/specialists/:id/tools", h.RenderToolsPage)
	rg.POST("/specialists/:id/tools", h.HandleAssociate)
	rg.DELETE("/specialists/:id/tools/:name", h.HandleDissociate)
}

func (h *ToolHandler) RenderToolsPage(c *gin.Context) {
	specialistID := c.Param("id")

	allTools := h.toolRegistry.Definitions()
	associated, _ := h.toolRepo.FindToolNamesBySpecialistID(c.Request.Context(), specialistID)

	associatedMap := make(map[string]bool)
	for _, name := range associated {
		associatedMap[name] = true
	}

	c.HTML(http.StatusOK, "specialist/tools.html", gin.H{
		"SpecialistID": specialistID,
		"AllTools":     allTools,
		"Associated":   associatedMap,
	})
}

func (h *ToolHandler) HandleAssociate(c *gin.Context) {
	specialistID := c.Param("id")
	toolName := c.PostForm("tool_name")

	if err := h.toolRepo.Associate(c.Request.Context(), specialistID, toolName); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Header("HX-Trigger", "tools-updated")
	c.Status(http.StatusOK)
}

func (h *ToolHandler) HandleDissociate(c *gin.Context) {
	specialistID := c.Param("id")
	toolName := c.Param("name")

	if err := h.toolRepo.Dissociate(c.Request.Context(), specialistID, toolName); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Header("HX-Trigger", "tools-updated")
	c.Status(http.StatusOK)
}
```

- [ ] **Step 2: Create the HTMX template**

```html
<!-- web/templates/specialist/tools.html -->
{{define "specialist/tools.html"}}
<div id="tools-section" hx-get="/admin/specialists/{{.SpecialistID}}/tools" hx-trigger="tools-updated from:body" hx-swap="outerHTML">
  <h3 class="text-lg font-semibold mb-4">Tools do Especialista</h3>

  <div class="space-y-2">
    {{range .AllTools}}
    <div class="flex items-center justify-between p-3 border rounded">
      <div>
        <span class="font-medium">{{.Name}}</span>
        <span class="text-sm text-gray-500 ml-2">{{.Description}}</span>
        <span class="text-xs px-2 py-1 rounded bg-gray-100 ml-2">{{.Category}}</span>
      </div>
      <div>
        {{if index $.Associated .Name}}
        <button
          hx-delete="/admin/specialists/{{$.SpecialistID}}/tools/{{.Name}}"
          hx-swap="none"
          class="btn btn-sm btn-outline btn-error">
          Remover
        </button>
        {{else}}
        <form hx-post="/admin/specialists/{{$.SpecialistID}}/tools" hx-swap="none">
          <input type="hidden" name="tool_name" value="{{.Name}}">
          <button type="submit" class="btn btn-sm btn-outline btn-success">
            Adicionar
          </button>
        </form>
        {{end}}
      </div>
    </div>
    {{end}}
  </div>
</div>
{{end}}
```

- [ ] **Step 3: Register routes in specialist handler**

In `internal/specialist/interfaces/http/handler.go`, in the `RegisterRoutes` method, add:

```go
// After existing specialist routes:
h.toolHandler.RegisterRoutes(adminGroup)
```

- [ ] **Step 4: Wire handler in specialist module**

In `internal/specialist/module.go`, create and wire the `ToolHandler`:

```go
toolHandler := specialisthttp.NewToolHandler(specialistToolRepo, toolRegistry)
```

Pass `toolRegistry` as a dependency from the AI module (or accept it in the specialist module constructor). The simplest approach: the specialist module receives the `ToolRegistry` as a parameter.

- [ ] **Step 5: Test manually in browser**

1. Start dev server: `air` or `go run cmd/api/main.go`
2. Navigate to `/admin/specialists/<id>/tools`
3. Verify tools listed with checkboxes
4. Add/remove tool associations
5. Verify HTMX partial reload works

- [ ] **Step 6: Commit**

```bash
git add internal/specialist/interfaces/http/tool_handler.go web/templates/specialist/tools.html internal/specialist/interfaces/http/handler.go internal/specialist/module.go
git commit -m "feat(F15): add admin UI for specialist tool association — handler, routes, HTMX template"
```

---

## Task 18: Integration Test — Full Tool Loop

**Files:**
- Create: `internal/ai/application/conversation_engine_tools_test.go`

- [ ] **Step 1: Write integration test**

```go
// internal/ai/application/conversation_engine_tools_test.go
package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeToolProviderWithLoop simulates a provider that first requests a tool call,
// then returns a text response using the tool result.
type fakeToolProviderWithLoop struct {
	callCount int
}

func (f *fakeToolProviderWithLoop) Name() string { return "fake" }

func (f *fakeToolProviderWithLoop) GenerateResponse(_ context.Context, req *domain.AIRequest) (*domain.AIResponse, error) {
	f.callCount++

	// First call: request search_leads tool
	if f.callCount == 1 {
		return &domain.AIResponse{
			FinishReason: "tool_calls",
			ToolCalls: []domain.ToolCall{
				{
					ID:       "call-1",
					ToolName: "search_leads",
					Arguments: map[string]interface{}{
						"query": "joao",
					},
				},
			},
			PromptTokens:     100,
			CompletionTokens: 20,
		}, nil
	}

	// Second call: use tool result in response
	if len(req.ToolResults) > 0 {
		return &domain.AIResponse{
			Content:          fmt.Sprintf("Encontrei resultados: %s", req.ToolResults[0].Content),
			FinishReason:     "stop",
			PromptTokens:     150,
			CompletionTokens: 30,
		}, nil
	}

	return &domain.AIResponse{Content: "no results", FinishReason: "stop"}, nil
}

// echoTool returns whatever query was passed as the result.
type echoTool struct{}

func (e *echoTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition("search_leads", "Search leads", domain.ToolCategoryDataQuery, map[string]domain.ParameterDef{
		"query": {Type: "string", Description: "Query", Required: true},
	})
	return def
}

func (e *echoTool) Execute(_ context.Context, _ string, args map[string]interface{}) (*domain.ToolResult, error) {
	query, _ := args["query"].(string)
	r := domain.NewToolResult("call-1", "Lead: Joao Silva (score 85) | query="+query, false)
	return &r, nil
}

func TestConversationEngine_ToolLoopIntegration(t *testing.T) {
	provider := &fakeToolProviderWithLoop{}
	registry := NewToolRegistry()
	registry.Register(&echoTool{})
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry:    registry,
		toolLoopMaxIter: 5,
		log:             log,
	}

	req := &domain.AIRequest{
		SystemPrompt: "Voce e um assistente",
		Messages:     []domain.AIMessage{{Role: domain.RoleUser, Content: "busca joao"}},
		Tools: []domain.ToolDefinition{
			registry.Definitions()[0],
		},
	}

	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Contains(t, resp.Content, "Encontrei resultados")
	assert.Contains(t, resp.Content, "Joao Silva")
	assert.Equal(t, 2, provider.callCount)
}

func TestConversationEngine_ToolLoopNoTools(t *testing.T) {
	provider := &fakeToolProvider{
		responses: []*domain.AIResponse{
			{Content: "Ola! Como posso ajudar?", FinishReason: "stop"},
		},
	}
	registry := NewToolRegistry()
	log := zap.NewNop()

	engine := &ConversationEngine{
		toolRegistry:    registry,
		toolLoopMaxIter: 5,
		log:             log,
	}

	req := &domain.AIRequest{
		Messages: []domain.AIMessage{{Role: domain.RoleUser, Content: "oi"}},
	}

	resp, err := engine.executeToolLoop(context.Background(), provider, req, "tenant-1", "spec-1", 5, 4000)
	require.NoError(t, err)
	assert.Equal(t, "Ola! Como posso ajudar?", resp.Content)
	assert.Equal(t, 1, provider.callCount)
}
```

- [ ] **Step 2: Run integration test**

Run: `go test ./internal/ai/application/ -run "TestConversationEngine_ToolLoop" -v`
Expected: PASS

- [ ] **Step 3: Run full test suite**

Run: `go test ./... -count=1`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/ai/application/conversation_engine_tools_test.go
git commit -m "test(F15): add integration tests for tool calling loop in ConversationEngine"
```

---

## Task 19: Final Verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 -race`
Expected: PASS with no race conditions

- [ ] **Step 2: Check test coverage**

Run: `go test ./internal/ai/... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total`
Expected: >= 80%

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/api/`
Expected: SUCCESS

- [ ] **Step 4: Run linter**

Run: `golangci-lint run ./...`
Expected: No errors

- [ ] **Step 5: Manual smoke test**

1. Start server with `air`
2. Go to `/admin/specialists/<id>/tools` — verify UI loads
3. Associate tools to a specialist
4. Send a WhatsApp message (or use playground) that triggers a tool call
5. Verify tool executes and specialist responds using the tool result

- [ ] **Step 6: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "fix(F15): address final verification issues"
```
