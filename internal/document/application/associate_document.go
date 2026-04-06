package application

import (
	"context"
	"errors"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

const MaxAssociationBatchSize = 50

var ErrAssociationBatchTooLarge = errors.New("cannot associate more than 50 documents at once")

type AssociateDocumentInput struct {
	SpecialistID string
	DocumentIDs  []string
}

type SpecialistFinder interface {
	FindByID(ctx context.Context, id string) (*specialistdomain.Specialist, error)
}

type AssociateDocumentUseCase struct {
	specFinder  SpecialistFinder
	docRepo     domain.DocumentRepository
	specDocRepo domain.SpecialistDocumentRepository
}

func NewAssociateDocumentUseCase(
	specFinder SpecialistFinder,
	docRepo domain.DocumentRepository,
	specDocRepo domain.SpecialistDocumentRepository,
) *AssociateDocumentUseCase {
	return &AssociateDocumentUseCase{
		specFinder:  specFinder,
		docRepo:     docRepo,
		specDocRepo: specDocRepo,
	}
}

func (uc *AssociateDocumentUseCase) Execute(ctx context.Context, input AssociateDocumentInput) error {
	if len(input.DocumentIDs) > MaxAssociationBatchSize {
		return ErrAssociationBatchTooLarge
	}

	specialist, err := uc.specFinder.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return err
	}
	if !specialist.IsActive() {
		return specialistdomain.ErrSpecialistInactive
	}

	for _, docID := range input.DocumentIDs {
		if _, err := uc.docRepo.FindByID(ctx, docID); err != nil {
			return err
		}

		exists, err := uc.specDocRepo.Exists(ctx, input.SpecialistID, docID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if err := uc.specDocRepo.Associate(ctx, input.SpecialistID, docID); err != nil {
			return err
		}
	}

	return nil
}
