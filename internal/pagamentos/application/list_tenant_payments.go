package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type ListTenantInput struct {
	TenantID    string
	Status      *domain.PaymentStatus
	DataInicial *time.Time
	DataFinal   *time.Time
	Page        int
	PageSize    int
}

type ListTenantPayments struct {
	repo domain.PaymentRepository
}

func NewListTenantPayments(repo domain.PaymentRepository) *ListTenantPayments {
	return &ListTenantPayments{repo: repo}
}

func (uc *ListTenantPayments) Execute(ctx context.Context, in ListTenantInput) (*domain.ListResult, error) {
	return uc.repo.List(ctx, domain.ListFilters{
		TenantID:    in.TenantID,
		Status:      in.Status,
		DataInicial: in.DataInicial,
		DataFinal:   in.DataFinal,
		Page:        in.Page,
		PageSize:    in.PageSize,
	})
}
