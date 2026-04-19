package application_test

import (
	"context"
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
	assert.Nil(t, fp.lastUserFilter) // owner não filtra
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
	require.NotNil(t, fp.lastUserFilter)
	assert.Equal(t, "u2", *fp.lastUserFilter)
	assert.Equal(t, "João", out.CurrentUserName)
}

func TestGetTenantDashboard_RejectsEmptyTenant(t *testing.T) {
	uc := application.NewGetTenantDashboard(newFakes(), fakeUserLookup{}, fixedClock{t: time.Now()})
	_, err := uc.Execute(context.Background(), application.TenantInput{TenantID: "", UserID: "u1", IsOwner: true})
	assert.ErrorIs(t, err, domain.ErrTenantRequired)
}
