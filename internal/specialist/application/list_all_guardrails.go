package application

import (
	"context"
	"sort"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ListAllGuardrailsUseCase lists the whole guardrail library (all reusable
// guardrails, regardless of attachment).
type ListAllGuardrailsUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewListAllGuardrailsUseCase(guardrailRepo domain.GuardrailRepository) *ListAllGuardrailsUseCase {
	return &ListAllGuardrailsUseCase{guardrailRepo: guardrailRepo}
}

func (uc *ListAllGuardrailsUseCase) Execute(ctx context.Context) ([]GuardrailOutput, error) {
	guardrails, err := uc.guardrailRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]GuardrailOutput, len(guardrails))
	for i := range guardrails {
		items[i] = newGuardrailOutput(&guardrails[i])
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}
