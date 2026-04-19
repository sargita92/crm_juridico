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

type fakeAdminProvider struct {
	tenants *domain.TenantsBlock
	usage   *domain.UsageBlock
	health  *domain.HealthBlock
	spec    *domain.SpecialistsBlock
	fin     *domain.FinancialBlock

	// injeção de erros — um por método para validar propagação
	errTenants error
	errUsage   error
	errHealth  error
	errSpec    error
	errFin     error
}

func (f *fakeAdminProvider) TenantsBlock(_ context.Context, _ time.Time) (*domain.TenantsBlock, error) {
	if f.errTenants != nil {
		return nil, f.errTenants
	}
	return f.tenants, nil
}
func (f *fakeAdminProvider) UsageBlock(_ context.Context) (*domain.UsageBlock, error) {
	if f.errUsage != nil {
		return nil, f.errUsage
	}
	return f.usage, nil
}
func (f *fakeAdminProvider) HealthBlock(_ context.Context, _ time.Time) (*domain.HealthBlock, error) {
	if f.errHealth != nil {
		return nil, f.errHealth
	}
	return f.health, nil
}
func (f *fakeAdminProvider) EspecialistasBlock(_ context.Context) (*domain.SpecialistsBlock, error) {
	if f.errSpec != nil {
		return nil, f.errSpec
	}
	return f.spec, nil
}
func (f *fakeAdminProvider) FinanceiroBlock(_ context.Context, _ time.Time) (*domain.FinancialBlock, error) {
	if f.errFin != nil {
		return nil, f.errFin
	}
	return f.fin, nil
}

type fakeInfraProvider struct {
	infra       *domain.Infrastructure
	errSnapshot error
}

func (f fakeInfraProvider) Snapshot(_ context.Context) (*domain.Infrastructure, error) {
	if f.errSnapshot != nil {
		return nil, f.errSnapshot
	}
	return f.infra, nil
}
