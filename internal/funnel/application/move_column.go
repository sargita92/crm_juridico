package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

var ErrInvalidDirection = errors.New("direction must be 'up' or 'down'")
var ErrCannotMoveColumn = errors.New("cannot move column in that direction")

type MoveColumnInput struct {
	TenantID  string
	FunnelID  string
	ColumnID  string
	Direction string // "up" or "down"
}

type MoveColumnUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
}

func NewMoveColumnUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository) *MoveColumnUseCase {
	return &MoveColumnUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo}
}

func (uc *MoveColumnUseCase) Execute(ctx context.Context, input MoveColumnInput) error {
	if input.Direction != "up" && input.Direction != "down" {
		return ErrInvalidDirection
	}

	funnel, err := uc.funnelRepo.FindByID(ctx, input.FunnelID)
	if err != nil {
		return err
	}
	if funnel.TenantID != input.TenantID {
		return domain.ErrFunnelNotFound
	}

	columns, err := uc.columnRepo.FindByFunnelID(ctx, input.FunnelID)
	if err != nil {
		return err
	}

	// Find target column index
	targetIdx := -1
	for i, c := range columns {
		if c.ID == input.ColumnID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		return domain.ErrColumnNotFound
	}

	// Find neighbor
	neighborIdx := -1
	if input.Direction == "up" {
		if targetIdx == 0 {
			return ErrCannotMoveColumn
		}
		neighborIdx = targetIdx - 1
	} else {
		if targetIdx == len(columns)-1 {
			return ErrCannotMoveColumn
		}
		neighborIdx = targetIdx + 1
	}

	target := columns[targetIdx]
	neighbor := columns[neighborIdx]

	return uc.columnRepo.SwapOrder(ctx, target.ID, neighbor.OrderIndex, neighbor.ID, target.OrderIndex)
}
