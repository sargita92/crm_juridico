package application_test

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

type fakeTenantProvider struct {
	funil     *domain.FunilBlock
	funilName string
	whats     *domain.WhatsAppStats
	resp      []domain.ResponsiblePerformance
	tempo     []domain.ColumnDwell
	prod      []domain.ProductLeadsCount

	// captura filtros passados — um por método para validar fan-out
	lastUserFilter      *string
	lastUserFilterWhats *string
	lastUserFilterResp  *string
	lastUserFilterTempo *string
	lastUserFilterProd  *string

	// injeção de erros — um por método para validar propagação
	errFunil error
	errWhats error
	errResp  error
	errTempo error
	errProd  error
}

func (f *fakeTenantProvider) FunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) (*domain.FunilBlock, string, error) {
	f.lastUserFilter = userID
	if f.errFunil != nil {
		return nil, "", f.errFunil
	}
	return f.funil, f.funilName, nil
}
func (f *fakeTenantProvider) WhatsAppBlock(ctx context.Context, _ string, userID *string) (*domain.WhatsAppStats, error) {
	f.lastUserFilterWhats = userID
	if f.errWhats != nil {
		return nil, f.errWhats
	}
	return f.whats, nil
}
func (f *fakeTenantProvider) ResponsaveisBlock(ctx context.Context, _ string, userID *string) ([]domain.ResponsiblePerformance, error) {
	f.lastUserFilterResp = userID
	if f.errResp != nil {
		return nil, f.errResp
	}
	return f.resp, nil
}
func (f *fakeTenantProvider) TempoFunilBlock(ctx context.Context, _ string, userID *string, now time.Time) ([]domain.ColumnDwell, error) {
	f.lastUserFilterTempo = userID
	if f.errTempo != nil {
		return nil, f.errTempo
	}
	return f.tempo, nil
}
func (f *fakeTenantProvider) ProdutosBlock(ctx context.Context, _ string, userID *string) ([]domain.ProductLeadsCount, error) {
	f.lastUserFilterProd = userID
	if f.errProd != nil {
		return nil, f.errProd
	}
	return f.prod, nil
}

type fakeUserLookup struct{ m map[string]string }

func (f fakeUserLookup) UserName(_ context.Context, userID string) (string, error) {
	if n, ok := f.m[userID]; ok {
		return n, nil
	}
	return "", nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }
