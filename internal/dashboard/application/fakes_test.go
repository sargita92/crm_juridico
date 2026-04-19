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

	// captura filtros passados
	lastUserFilter *string
}

func (f *fakeTenantProvider) FunilBlock(ctx context.Context, tenantID string, userID *string, now time.Time) (*domain.FunilBlock, string, error) {
	f.lastUserFilter = userID
	return f.funil, f.funilName, nil
}
func (f *fakeTenantProvider) WhatsAppBlock(ctx context.Context, _ string, userID *string) (*domain.WhatsAppStats, error) {
	return f.whats, nil
}
func (f *fakeTenantProvider) ResponsaveisBlock(ctx context.Context, _ string, userID *string) ([]domain.ResponsiblePerformance, error) {
	return f.resp, nil
}
func (f *fakeTenantProvider) TempoFunilBlock(ctx context.Context, _ string, userID *string, now time.Time) ([]domain.ColumnDwell, error) {
	return f.tempo, nil
}
func (f *fakeTenantProvider) ProdutosBlock(ctx context.Context, _ string, userID *string) ([]domain.ProductLeadsCount, error) {
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
