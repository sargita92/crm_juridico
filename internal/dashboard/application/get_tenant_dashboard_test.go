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

func newFakes() *fakeTenantProvider {
	return &fakeTenantProvider{
		funil:     &domain.FunilBlock{StatusTotals: domain.LeadStatusCount{Open: 3, Won: 2, Lost: 1}, ConversionPct: 66.66},
		funilName: "Funil Comercial",
		whats:     &domain.WhatsAppStats{IncomingMessages: 10, OutgoingMessages: 8, ActiveConversations: 4},
		resp:      []domain.ResponsiblePerformance{{UserID: "u1", UserName: "Maria", Total: 5, Won: 2, Lost: 1}},
		tempo:     []domain.ColumnDwell{{ColumnID: "c1", ColumnName: "Novos", AvgHours: 12, StuckOver7Days: 0}},
		prod:      []domain.ProductLeadsCount{{ProductID: "p1", ProductName: "Consulta", Total: 4, Won: 1}},
	}
}

func TestGetTenantDashboard_Owner_SeesTenantWideData(t *testing.T) {
	fp := newFakes()
	uc := application.NewGetTenantDashboard(fp, fakeUserLookup{}, fixedClock{t: time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC)})
	out, err := uc.Execute(context.Background(), application.TenantInput{
		TenantID: "t1", UserID: "u1", IsOwner: true,
	})
	require.NoError(t, err)
	assert.False(t, out.ScopeIsUser)
	// owner não filtra — fan-out: todos os 5 métodos recebem nil
	assert.Nil(t, fp.lastUserFilter)
	assert.Nil(t, fp.lastUserFilterWhats)
	assert.Nil(t, fp.lastUserFilterResp)
	assert.Nil(t, fp.lastUserFilterTempo)
	assert.Nil(t, fp.lastUserFilterProd)
	assert.Equal(t, int64(3), out.Bloco1_Funil.StatusTotals.Open)
}

func TestGetTenantDashboard_CommonUser_FiltersByResponsible(t *testing.T) {
	fp := newFakes()
	ul := fakeUserLookup{m: map[string]string{"u2": "João"}}
	uc := application.NewGetTenantDashboard(fp, ul, fixedClock{t: time.Now()})
	out, err := uc.Execute(context.Background(), application.TenantInput{
		TenantID: "t1", UserID: "u2", IsOwner: false,
	})
	require.NoError(t, err)
	assert.True(t, out.ScopeIsUser)
	// fan-out: todos os 5 métodos devem receber o mesmo filtro "u2"
	require.NotNil(t, fp.lastUserFilter)
	require.NotNil(t, fp.lastUserFilterWhats)
	require.NotNil(t, fp.lastUserFilterResp)
	require.NotNil(t, fp.lastUserFilterTempo)
	require.NotNil(t, fp.lastUserFilterProd)
	assert.Equal(t, "u2", *fp.lastUserFilter)
	assert.Equal(t, "u2", *fp.lastUserFilterWhats)
	assert.Equal(t, "u2", *fp.lastUserFilterResp)
	assert.Equal(t, "u2", *fp.lastUserFilterTempo)
	assert.Equal(t, "u2", *fp.lastUserFilterProd)
	assert.Equal(t, "João", out.CurrentUserName)
}

func TestGetTenantDashboard_RejectsEmptyTenant(t *testing.T) {
	uc := application.NewGetTenantDashboard(newFakes(), fakeUserLookup{}, fixedClock{t: time.Now()})
	_, err := uc.Execute(context.Background(), application.TenantInput{TenantID: "", UserID: "u1", IsOwner: true})
	assert.ErrorIs(t, err, domain.ErrTenantRequired)
}

func TestGetTenantDashboard_RejectsEmptyUser(t *testing.T) {
	uc := application.NewGetTenantDashboard(newFakes(), fakeUserLookup{}, fixedClock{t: time.Now()})
	_, err := uc.Execute(context.Background(), application.TenantInput{TenantID: "t1", UserID: "", IsOwner: true})
	assert.ErrorIs(t, err, domain.ErrUserRequired)
}

func TestGetTenantDashboard_PropagatesProviderError(t *testing.T) {
	sentinel := errors.New("boom")

	cases := []struct {
		name   string
		inject func(fp *fakeTenantProvider)
	}{
		{"funil", func(fp *fakeTenantProvider) { fp.errFunil = sentinel }},
		{"whatsapp", func(fp *fakeTenantProvider) { fp.errWhats = sentinel }},
		{"responsive", func(fp *fakeTenantProvider) { fp.errResp = sentinel }},
		{"tempo_funil", func(fp *fakeTenantProvider) { fp.errTempo = sentinel }},
		{"produtos", func(fp *fakeTenantProvider) { fp.errProd = sentinel }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fp := newFakes()
			tc.inject(fp)
			uc := application.NewGetTenantDashboard(fp, fakeUserLookup{}, fixedClock{t: time.Now()})
			out, err := uc.Execute(context.Background(), application.TenantInput{
				TenantID: "t1", UserID: "u1", IsOwner: true,
			})
			assert.ErrorIs(t, err, sentinel)
			assert.Nil(t, out)
		})
	}
}
