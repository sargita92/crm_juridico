package domain

import (
	"context"
	"time"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type ListFilters struct {
	TenantID    string
	Status      *PaymentStatus
	DataInicial *time.Time
	DataFinal   *time.Time
	Page        int
	PageSize    int
}

type ListResult struct {
	Items    []Payment
	Total    int64
	Page     int
	PageSize int
}

type Summary struct {
	TotalPagoAnoCents  int64
	TotalPendenteCents int64
	TotalAtrasadoCents int64
	HasAtrasado        bool
	HasPendente        bool
}

type PaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	Update(ctx context.Context, p *Payment) error
	FindByID(ctx context.Context, tenantID, id string) (*Payment, error)
	FindByIDAdmin(ctx context.Context, id string) (*Payment, error)
	ExistsRecorrente(ctx context.Context, tenantID string, competencia time.Time) (bool, error)
	List(ctx context.Context, f ListFilters) (*ListResult, error)
	ListAll(ctx context.Context, f ListFilters) (*ListResult, error)
	ListOverdueCandidates(ctx context.Context, today time.Time) ([]Payment, error)
	Summary(ctx context.Context, tenantID string, today time.Time) (*Summary, error)
}
