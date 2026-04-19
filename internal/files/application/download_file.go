package application

import (
	"context"
	"io"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type DownloadFileUseCase struct {
	repo    domain.FileRepository
	storage domain.Storage
}

func NewDownloadFileUseCase(repo domain.FileRepository, storage domain.Storage) *DownloadFileUseCase {
	return &DownloadFileUseCase{repo: repo, storage: storage}
}

func (uc *DownloadFileUseCase) Execute(ctx context.Context, tenantID, id string) (*domain.File, io.ReadCloser, error) {
	if tenantID == "" {
		return nil, nil, domain.ErrTenantIDRequired
	}
	f, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	rc, err := uc.storage.Open(ctx, f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return f, rc, nil
}
