package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

func newAdminFakes() (*fakeAdminProvider, fakeInfraProvider) {
	ap := &fakeAdminProvider{
		tenants: &domain.TenantsBlock{Totals: domain.TenantStatusCount{Active: 10, Inactive: 2, Blocked: 1}, NewThisMonth: 3},
		usage:   &domain.UsageBlock{TotalLeads: 500, TotalMessages: 2000, ActiveConversations: 50},
		health:  &domain.HealthBlock{},
		spec:    &domain.SpecialistsBlock{Total: 8, Qualifications: 120},
		fin:     &domain.FinancialBlock{ReceitaAnoCents: 1_500_000, PendenteTotalCents: 200_000, AtrasadoTotalCents: 50_000},
	}
	ip := fakeInfraProvider{infra: &domain.Infrastructure{APILatencyMs: 120.5, Error5xxRate: 0.001}}
	return ap, ip
}

func TestGetAdminDashboard_ReturnsAllBlocks(t *testing.T) {
	ap, ip := newAdminFakes()
	uc := application.NewGetAdminDashboard(ap, ip, fixedClock{t: time.Now()})
	out, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(10), out.Bloco1_Tenants.Totals.Active)
	assert.Equal(t, int64(500), out.Bloco2_Uso.TotalLeads)
	assert.Equal(t, int64(8), out.Bloco5_Especialistas.Total)
	assert.Equal(t, int64(1_500_000), out.Bloco6_Financeiro.ReceitaAnoCents)
	assert.Equal(t, 120.5, out.Bloco4_Infra.APILatencyMs)
}

func TestGetAdminDashboard_PropagatesProviderError(t *testing.T) {
	sentinel := errors.New("boom")

	cases := []struct {
		name   string
		inject func(ap *fakeAdminProvider, ip *fakeInfraProvider)
	}{
		{"tenants", func(ap *fakeAdminProvider, _ *fakeInfraProvider) { ap.errTenants = sentinel }},
		{"usage", func(ap *fakeAdminProvider, _ *fakeInfraProvider) { ap.errUsage = sentinel }},
		{"health", func(ap *fakeAdminProvider, _ *fakeInfraProvider) { ap.errHealth = sentinel }},
		{"especialistas", func(ap *fakeAdminProvider, _ *fakeInfraProvider) { ap.errSpec = sentinel }},
		{"financeiro", func(ap *fakeAdminProvider, _ *fakeInfraProvider) { ap.errFin = sentinel }},
		{"infra", func(_ *fakeAdminProvider, ip *fakeInfraProvider) { ip.errSnapshot = sentinel }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ap, ip := newAdminFakes()
			tc.inject(ap, &ip)
			uc := application.NewGetAdminDashboard(ap, ip, fixedClock{t: time.Now()})
			out, err := uc.Execute(context.Background())
			assert.ErrorIs(t, err, sentinel)
			assert.Nil(t, out)
		})
	}
}
