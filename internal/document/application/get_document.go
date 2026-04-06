package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type DocumentDetailOutput struct {
	ID              string
	Name            string
	Type            string
	FilePath        string
	FileSize        int64
	SpecialistCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type GetDocumentUseCase struct {
	repo        domain.DocumentRepository
	specDocRepo domain.SpecialistDocumentRepository
}

func NewGetDocumentUseCase(repo domain.DocumentRepository, specDocRepo domain.SpecialistDocumentRepository) *GetDocumentUseCase {
	return &GetDocumentUseCase{repo: repo, specDocRepo: specDocRepo}
}

func (uc *GetDocumentUseCase) Execute(ctx context.Context, id string) (*DocumentDetailOutput, error) {
	doc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	count, err := uc.specDocRepo.CountByDocumentID(ctx, doc.ID)
	if err != nil {
		return nil, err
	}

	return &DocumentDetailOutput{
		ID:              doc.ID,
		Name:            doc.Name,
		Type:            doc.Type,
		FilePath:        doc.FilePath,
		FileSize:        doc.FileSize,
		SpecialistCount: count,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}, nil
}
