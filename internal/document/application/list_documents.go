package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type ListDocumentsInput struct {
	Search string
	Page   int
	Limit  int
}

type DocumentItem struct {
	ID              string
	Name            string
	Type            string
	FileSize        int64
	SpecialistCount int
	CreatedAt       time.Time
}

type ListDocumentsOutput struct {
	Documents []DocumentItem
	Total     int64
	Page      int
	Limit     int
}

type ListDocumentsUseCase struct {
	repo       domain.DocumentRepository
	specDocRepo domain.SpecialistDocumentRepository
}

func NewListDocumentsUseCase(repo domain.DocumentRepository, specDocRepo domain.SpecialistDocumentRepository) *ListDocumentsUseCase {
	return &ListDocumentsUseCase{repo: repo, specDocRepo: specDocRepo}
}

func (uc *ListDocumentsUseCase) Execute(ctx context.Context, input ListDocumentsInput) (*ListDocumentsOutput, error) {
	filter := domain.DocumentFilter{
		Search: input.Search,
		Page:   input.Page,
		Limit:  input.Limit,
	}

	list, err := uc.repo.FindWithFilter(ctx, filter)
	if err != nil {
		return nil, err
	}

	items := make([]DocumentItem, len(list.Documents))
	for i, doc := range list.Documents {
		count, err := uc.specDocRepo.CountByDocumentID(ctx, doc.ID)
		if err != nil {
			return nil, err
		}
		items[i] = DocumentItem{
			ID:              doc.ID,
			Name:            doc.Name,
			Type:            doc.Type,
			FileSize:        doc.FileSize,
			SpecialistCount: count,
			CreatedAt:       doc.CreatedAt,
		}
	}

	return &ListDocumentsOutput{
		Documents: items,
		Total:     list.Total,
		Page:      list.Page,
		Limit:     list.Limit,
	}, nil
}
