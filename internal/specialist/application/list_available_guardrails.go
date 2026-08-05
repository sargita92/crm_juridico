package application

import (
	"context"
	"sort"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ListAvailableGuardrailsUseCase lists library guardrails that are NOT yet
// attached to the given specialist — the pick list for the "attach existing"
// action on the specialist detail page.
type ListAvailableGuardrailsUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewListAvailableGuardrailsUseCase(guardrailRepo domain.GuardrailRepository) *ListAvailableGuardrailsUseCase {
	return &ListAvailableGuardrailsUseCase{guardrailRepo: guardrailRepo}
}

func (uc *ListAvailableGuardrailsUseCase) Execute(ctx context.Context, specialistID string) ([]GuardrailOutput, error) {
	all, err := uc.guardrailRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	attached, err := uc.guardrailRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	attachedIDs := make(map[string]bool, len(attached))
	for _, g := range attached {
		attachedIDs[g.ID] = true
	}

	var items []GuardrailOutput
	for i := range all {
		if !attachedIDs[all[i].ID] {
			items = append(items, newGuardrailOutput(&all[i]))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
