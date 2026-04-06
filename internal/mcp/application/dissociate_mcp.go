package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

type DissociateMcpUseCase struct {
	specMcpRepo domain.SpecialistMcpRepository
}

func NewDissociateMcpUseCase(specMcpRepo domain.SpecialistMcpRepository) *DissociateMcpUseCase {
	return &DissociateMcpUseCase{specMcpRepo: specMcpRepo}
}

func (uc *DissociateMcpUseCase) Execute(ctx context.Context, specialistID, mcpID string) error {
	return uc.specMcpRepo.Dissociate(ctx, specialistID, mcpID)
}
