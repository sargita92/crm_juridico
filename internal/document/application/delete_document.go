package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type DeleteDocumentUseCase struct {
	repo        domain.DocumentRepository
	specDocRepo domain.SpecialistDocumentRepository
}

func NewDeleteDocumentUseCase(repo domain.DocumentRepository, specDocRepo domain.SpecialistDocumentRepository) *DeleteDocumentUseCase {
	return &DeleteDocumentUseCase{repo: repo, specDocRepo: specDocRepo}
}

func (uc *DeleteDocumentUseCase) Execute(ctx context.Context, id string) error {
	if _, err := uc.repo.FindByID(ctx, id); err != nil {
		return err
	}

	if err := uc.specDocRepo.DissociateAllByDocumentID(ctx, id); err != nil {
		return err
	}

	return uc.repo.Delete(ctx, id)
}
