package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type ListSpecialistDocumentsUseCase struct {
	specDocRepo domain.SpecialistDocumentRepository
	docRepo     domain.DocumentRepository
}

func NewListSpecialistDocumentsUseCase(specDocRepo domain.SpecialistDocumentRepository, docRepo domain.DocumentRepository) *ListSpecialistDocumentsUseCase {
	return &ListSpecialistDocumentsUseCase{specDocRepo: specDocRepo, docRepo: docRepo}
}

func (uc *ListSpecialistDocumentsUseCase) Execute(ctx context.Context, specialistID string) ([]DocumentItem, error) {
	docIDs, err := uc.specDocRepo.FindDocumentIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	if len(docIDs) == 0 {
		return []DocumentItem{}, nil
	}

	docs, err := uc.docRepo.FindByIDs(ctx, docIDs)
	if err != nil {
		return nil, err
	}

	items := make([]DocumentItem, len(docs))
	for i, doc := range docs {
		items[i] = DocumentItem{
			ID:       doc.ID,
			Name:     doc.Name,
			Type:     doc.Type,
			FileSize: doc.FileSize,
		}
	}
	return items, nil
}
