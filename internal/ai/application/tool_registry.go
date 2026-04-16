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
