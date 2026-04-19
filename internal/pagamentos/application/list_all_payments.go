package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type ListAllInput struct {
	TenantID    string
	Status      *domain.PaymentStatus
	DataInicial *time.Time
	DataFinal   *time.Time
	Page        int
	PageSize    int
}

type ListAllPayments struct {
	repo domain.PaymentRepository
}

func NewListAllPayments(repo domain.PaymentRepository) *ListAllPayments {
	return &ListAllPayments{repo: repo}
}

func (uc *ListAllPayments) Execute(ctx context.Context, in ListAllInput) (*domain.ListResult, error) {
	return uc.repo.ListAll(ctx, domain.ListFilters{
		TenantID:    in.TenantID,
		Status:      in.Status,
		DataInicial: in.DataInicial,
		DataFinal:   in.DataFinal,
		Page:        in.Page,
		PageSize:    in.PageSize,
	})
}
