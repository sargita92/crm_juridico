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

// ToolCategory classifies the intent/scope of a tool.
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

// ParameterDef describes a single input parameter for a tool.
type ParameterDef struct {
	// Type is the JSON Schema primitive type. Allowed values: "string", "number", "boolean".
	Type        string
	Description string
	Required    bool
	Enum        []string
}

// ToolDefinition is the static metadata that describes a callable tool.
type ToolDefinition struct {
	Name        string
	Description string
	Category    ToolCategory
	Parameters  map[string]ParameterDef
}

// NewToolDefinition constructs a ToolDefinition with validation.
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
// Instances are constructed by provider adapters (e.g., OpenAIProvider.parseToolCalls),
// not by application code.
type ToolCall struct {
	ID        string
	ToolName  string
	Arguments map[string]interface{}
}

// ToolResult holds the outcome of executing a ToolCall.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// NewToolResult creates a ToolResult without any content length restriction.
func NewToolResult(toolCallID, content string, isError bool) ToolResult {
	return ToolResult{ToolCallID: toolCallID, Content: content, IsError: isError}
}

// NewToolResultWithLimit creates a ToolResult, truncating content to maxLen bytes if necessary.
// Pass maxLen <= 0 to disable truncation.
func NewToolResultWithLimit(toolCallID, content string, isError bool, maxLen int) ToolResult {
	if maxLen > 0 && len(content) > maxLen {
		content = content[:maxLen]
	}
	return ToolResult{ToolCallID: toolCallID, Content: content, IsError: isError}
}

// Tool is the interface every executable tool must satisfy.
type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, tenantID string, args map[string]interface{}) (*ToolResult, error)
}
