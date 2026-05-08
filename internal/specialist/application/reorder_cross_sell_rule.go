package application

import (
	"context"
	"errors"
	"sort"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

var (
	ErrCrossSellRuleAlreadyFirst = errors.New("cross-sell rule is already at the first position")
	ErrCrossSellRuleAlreadyLast  = errors.New("cross-sell rule is already at the last position")
	ErrCrossSellMoveInvalid      = errors.New("direction must be 'up' or 'down'")
)

// ReorderCrossSellRuleInput holds the rule ID and direction to move.
type ReorderCrossSellRuleInput struct {
	ID        string
	Direction string // "up" or "down"
}

// ReorderCrossSellRuleUseCase swaps a rule's Ordem with its adjacent neighbor.
type ReorderCrossSellRuleUseCase struct {
	repo domain.CrossSellRuleRepository
}

func NewReorderCrossSellRuleUseCase(repo domain.CrossSellRuleRepository) *ReorderCrossSellRuleUseCase {
	return &ReorderCrossSellRuleUseCase{repo: repo}
}

func (uc *ReorderCrossSellRuleUseCase) Execute(ctx context.Context, input ReorderCrossSellRuleInput) error {
	if input.Direction != "up" && input.Direction != "down" {
		return ErrCrossSellMoveInvalid
	}

	rule, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	rules, err := uc.repo.ListBySpecialistID(ctx, rule.SpecialistID)
	if err != nil {
		return err
	}

	// Sort by Ordem ascending
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Ordem < rules[j].Ordem
	})

	currentIdx := -1
	for i, r := range rules {
		if r.ID == rule.ID {
			currentIdx = i
			break
		}
	}

	if input.Direction == "up" {
		if currentIdx == 0 {
			return ErrCrossSellRuleAlreadyFirst
		}
		neighbor := rules[currentIdx-1]
		// Swap Ordem
		rule.Ordem, neighbor.Ordem = neighbor.Ordem, rule.Ordem
		if err := uc.repo.Save(ctx, neighbor); err != nil {
			return err
		}
		return uc.repo.Save(ctx, rule)
	}

	// down
	if currentIdx == len(rules)-1 {
		return ErrCrossSellRuleAlreadyLast
	}
	neighbor := rules[currentIdx+1]
	rule.Ordem, neighbor.Ordem = neighbor.Ordem, rule.Ordem
	if err := uc.repo.Save(ctx, neighbor); err != nil {
		return err
	}
	return uc.repo.Save(ctx, rule)
}
