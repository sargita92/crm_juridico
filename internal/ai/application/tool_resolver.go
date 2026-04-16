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

func NewToolResolver(registry *ToolRegistry, finder SpecialistToolFinder) *ToolResolver {
	return &ToolResolver{registry: registry, finder: finder}
}

// ResolveForSpecialist returns the tools available for a given specialist.
// Tool names in DB but missing from the registry are silently skipped (e.g., removed tools).
func (r *ToolResolver) ResolveForSpecialist(ctx context.Context, specialistID string) ([]domain.Tool, error) {
	names, err := r.finder.FindToolNamesBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	var tools []domain.Tool
	for _, name := range names {
		tool, tErr := r.registry.Get(name)
		if tErr != nil {
			continue
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

// ApplyStepConstraints filters tools based on the current step's ForcedTools and RestrictedTools.
// - ForcedTools (if non-empty): keep only tools whose name is in the list (intersection).
// - RestrictedTools (if non-empty): remove tools whose name is in the list.
// Both constraints combine: Forced applies first, then Restricted is subtracted.
func (r *ToolResolver) ApplyStepConstraints(tools []domain.Tool, step *specDomain.Step) []domain.Tool {
	if step == nil {
		return tools
	}

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

// ResolveDefinitions is a convenience method that resolves for specialist, applies step constraints,
// and returns the ToolDefinitions (what AIRequest.Tools expects).
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
