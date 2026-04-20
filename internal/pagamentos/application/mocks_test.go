package application_test

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

var errBoom = errors.New("boom")

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time {
	if f.now.IsZero() {
		return time.Now()
	}
	return f.now
}

type fakeIDGen struct{ id string }

func (f *fakeIDGen) NewID() string {
	if f.id == "" {
		return "gen-default"
	}
	return f.id
}

type seqIDGen struct {
	prefix string
	seq    int
}

func (s *seqIDGen) NewID() string {
	s.seq++
	return s.prefix + strconv.Itoa(s.seq)
}

type fakeRepo struct {
	payments           map[string]*domain.Payment
	createErr          error
	updateErr          error
	existsErr          error
	listResult         *domain.ListResult
	listErr            error
	listAllResult      *domain.ListResult
	listAllErr         error
	overdueResult      []domain.Payment
	overdueErr         error
	summaryResult      *domain.Summary
	summaryErr         error
	lastListFilters    domain.ListFilters
	lastListAllFilters domain.ListFilters
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{payments: map[string]*domain.Payment{}}
}

func (r *fakeRepo) Create(_ context.Context, p *domain.Payment) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.payments[p.ID] = p
	return nil
}

func (r *fakeRepo) Update(_ context.Context, p *domain.Payment) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.payments[p.ID] = p
	return nil
}

func (r *fakeRepo) FindByID(_ context.Context, tenantID, id string) (*domain.Payment, error) {
	if p, ok := r.payments[id]; ok && p.TenantID == tenantID {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (r *fakeRepo) FindByIDAdmin(_ context.Context, id string) (*domain.Payment, error) {
	if p, ok := r.payments[id]; ok {
		return p, nil
	}
	return nil, domain.ErrPaymentNotFound
}

func (r *fakeRepo) ExistsRecorrente(_ context.Context, tenantID string, comp time.Time) (bool, error) {
	if r.existsErr != nil {
		return false, r.existsErr
	}
	for _, p := range r.payments {
		if p.TenantID == tenantID && p.Tipo == domain.TypeRecorrente && p.Competencia != nil && p.Competencia.Equal(comp) {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRepo) List(_ context.Context, f domain.ListFilters) (*domain.ListResult, error) {
	r.lastListFilters = f
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listResult, nil
}

func (r *fakeRepo) ListAll(_ context.Context, f domain.ListFilters) (*domain.ListResult, error) {
	r.lastListAllFilters = f
	if r.listAllErr != nil {
		return nil, r.listAllErr
	}
	return r.listAllResult, nil
}

func (r *fakeRepo) ListOverdueCandidates(_ context.Context, _ time.Time) ([]domain.Payment, error) {
	if r.overdueErr != nil {
		return nil, r.overdueErr
	}
	return r.overdueResult, nil
}

func (r *fakeRepo) Summary(_ context.Context, _ string, _ time.Time) (*domain.Summary, error) {
	if r.summaryErr != nil {
		return nil, r.summaryErr
	}
	return r.summaryResult, nil
}

func (r *fakeRepo) GlobalSummary(_ context.Context, _ time.Time) (*domain.GlobalSummary, error) {
	return &domain.GlobalSummary{}, nil
}

type fakeBillingRepo struct {
	byID   map[string]*domain.TenantBilling
	listed []domain.TenantBilling
	getErr error
}

func newFakeBillingRepo() *fakeBillingRepo {
	return &fakeBillingRepo{byID: map[string]*domain.TenantBilling{}}
}

func (r *fakeBillingRepo) GetByID(_ context.Context, tenantID string) (*domain.TenantBilling, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if tb, ok := r.byID[tenantID]; ok {
		return tb, nil
	}
	return nil, domain.ErrTenantNotFound
}

func (r *fakeBillingRepo) ListActiveBillable(_ context.Context) ([]domain.TenantBilling, error) {
	return r.listed, nil
}
