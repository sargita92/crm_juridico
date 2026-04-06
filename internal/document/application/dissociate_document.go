package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type DissociateDocumentUseCase struct {
	specDocRepo domain.SpecialistDocumentRepository
}

func NewDissociateDocumentUseCase(specDocRepo domain.SpecialistDocumentRepository) *DissociateDocumentUseCase {
	return &DissociateDocumentUseCase{specDocRepo: specDocRepo}
}

func (uc *DissociateDocumentUseCase) Execute(ctx context.Context, specialistID, documentID string) error {
	return uc.specDocRepo.Dissociate(ctx, specialistID, documentID)
}
