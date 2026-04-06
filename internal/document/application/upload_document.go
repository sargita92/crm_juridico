package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

type UploadDocumentInput struct {
	Name     string
	Type     string
	FilePath string
	FileSize int64
}

type UploadDocumentOutput struct {
	ID   string
	Name string
	Type string
}

type UploadDocumentUseCase struct {
	repo domain.DocumentRepository
}

func NewUploadDocumentUseCase(repo domain.DocumentRepository) *UploadDocumentUseCase {
	return &UploadDocumentUseCase{repo: repo}
}

func (uc *UploadDocumentUseCase) Execute(ctx context.Context, input UploadDocumentInput) (*UploadDocumentOutput, error) {
	doc, err := domain.NewDocument(uuid.New().String(), input.Name, input.Type, input.FilePath, input.FileSize)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, doc); err != nil {
		return nil, err
	}

	return &UploadDocumentOutput{
		ID:   doc.ID,
		Name: doc.Name,
		Type: doc.Type,
	}, nil
}
