package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type ListAvailableDocumentsUseCase struct {
	specDocRepo domain.SpecialistDocumentRepository
	docRepo     domain.DocumentRepository
}

func NewListAvailableDocumentsUseCase(specDocRepo domain.SpecialistDocumentRepository, docRepo domain.DocumentRepository) *ListAvailableDocumentsUseCase {
	return &ListAvailableDocumentsUseCase{specDocRepo: specDocRepo, docRepo: docRepo}
}

func (uc *ListAvailableDocumentsUseCase) Execute(ctx context.Context, specialistID string, search string) ([]DocumentItem, error) {
	associatedIDs, err := uc.specDocRepo.FindDocumentIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	associatedSet := make(map[string]bool, len(associatedIDs))
	for _, id := range associatedIDs {
		associatedSet[id] = true
	}

	list, err := uc.docRepo.FindWithFilter(ctx, domain.DocumentFilter{
		Search: search,
		Page:   1,
		Limit:  100,
	})
	if err != nil {
		return nil, err
	}

	var items []DocumentItem
	for _, doc := range list.Documents {
		if associatedSet[doc.ID] {
			continue
		}
		items = append(items, DocumentItem{
			ID:       doc.ID,
			Name:     doc.Name,
			Type:     doc.Type,
			FileSize: doc.FileSize,
		})
	}

	if items == nil {
		items = []DocumentItem{}
	}
	return items, nil
}
