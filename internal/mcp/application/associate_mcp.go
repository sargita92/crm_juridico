package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

const MaxAssociationBatchSize = 50

var ErrAssociationBatchTooLarge = errors.New("cannot associate more than 50 mcp servers at once")

type AssociateMcpInput struct {
	SpecialistID string
	McpIDs       []string
}

type SpecialistFinder interface {
	FindByID(ctx context.Context, id string) (*specialistdomain.Specialist, error)
}

type AssociateMcpUseCase struct {
	specFinder  SpecialistFinder
	mcpRepo     domain.McpServerRepository
	specMcpRepo domain.SpecialistMcpRepository
}

func NewAssociateMcpUseCase(
	specFinder SpecialistFinder,
	mcpRepo domain.McpServerRepository,
	specMcpRepo domain.SpecialistMcpRepository,
) *AssociateMcpUseCase {
	return &AssociateMcpUseCase{
		specFinder:  specFinder,
		mcpRepo:     mcpRepo,
		specMcpRepo: specMcpRepo,
	}
}

func (uc *AssociateMcpUseCase) Execute(ctx context.Context, input AssociateMcpInput) error {
	if len(input.McpIDs) > MaxAssociationBatchSize {
		return ErrAssociationBatchTooLarge
	}

	specialist, err := uc.specFinder.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return err
	}
	if !specialist.IsActive() {
		return specialistdomain.ErrSpecialistInactive
	}

	for _, mcpID := range input.McpIDs {
		if _, err := uc.mcpRepo.FindByID(ctx, mcpID); err != nil {
			return err
		}

		exists, err := uc.specMcpRepo.Exists(ctx, input.SpecialistID, mcpID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if err := uc.specMcpRepo.Associate(ctx, input.SpecialistID, mcpID); err != nil {
			return err
		}
	}

	return nil
}
