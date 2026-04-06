package application

import (
	"context"
	"errors"
	"sort"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

var (
	ErrMoveDirectionInvalid = errors.New("direction must be 'up' or 'down'")
	ErrStepAlreadyFirst     = errors.New("step is already at the first position")
	ErrStepAlreadyLast      = errors.New("step is already at the last position")
)

type MoveStepInput struct {
	ID        string
	Direction string
}

type MoveStepUseCase struct {
	stepRepo domain.StepRepository
}

func NewMoveStepUseCase(stepRepo domain.StepRepository) *MoveStepUseCase {
	return &MoveStepUseCase{stepRepo: stepRepo}
}

func (uc *MoveStepUseCase) Execute(ctx context.Context, input MoveStepInput) error {
	if input.Direction != "up" && input.Direction != "down" {
		return ErrMoveDirectionInvalid
	}

	s, err := uc.stepRepo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	steps, err := uc.stepRepo.FindBySpecialistID(ctx, s.SpecialistID)
	if err != nil {
		return err
	}

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].OrderIndex < steps[j].OrderIndex
	})

	currentIdx := -1
	for i, st := range steps {
		if st.ID == s.ID {
			currentIdx = i
			break
		}
	}

	if input.Direction == "up" {
		if currentIdx == 0 {
			return ErrStepAlreadyFirst
		}
		neighbor := steps[currentIdx-1]
		return uc.stepRepo.SwapOrder(ctx, s.ID, s.OrderIndex, neighbor.ID, neighbor.OrderIndex)
	}

	// down
	if currentIdx == len(steps)-1 {
		return ErrStepAlreadyLast
	}
	neighbor := steps[currentIdx+1]
	return uc.stepRepo.SwapOrder(ctx, s.ID, s.OrderIndex, neighbor.ID, neighbor.OrderIndex)
}
