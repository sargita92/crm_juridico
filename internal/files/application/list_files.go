package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type ListFilesInput struct {
	TenantID  string
	LeadID    *string
	MediaType *domain.MediaType
	Search    string
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}

type ListFilesUseCase struct {
	repo domain.FileRepository
}

func NewListFilesUseCase(repo domain.FileRepository) *ListFilesUseCase {
	return &ListFilesUseCase{repo: repo}
}

func (uc *ListFilesUseCase) Execute(ctx context.Context, in ListFilesInput) (*domain.ListResult, error) {
	if in.TenantID == "" {
		return nil, domain.ErrTenantIDRequired
	}
	if in.From != nil && in.To != nil && in.From.After(*in.To) {
		return nil, domain.ErrInvalidDateRange
	}
	if in.MediaType != nil && !domain.IsValidMediaType(*in.MediaType) {
		return nil, domain.ErrInvalidMediaType
	}

	page := in.Page
	if page < 1 {
		page = 1
	}
	size := in.PageSize
	if size <= 0 {
		size = domain.DefaultPageSize
	}
	if size > domain.MaxPageSize {
		size = domain.MaxPageSize
	}

	return uc.repo.List(ctx, domain.ListQuery{
		TenantID:  in.TenantID,
		LeadID:    in.LeadID,
		MediaType: in.MediaType,
		Search:    in.Search,
		From:      in.From,
		To:        in.To,
		Page:      page,
		PageSize:  size,
	})
}
