package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
	dashhttp "github.com/sasrgita/crm-juridico/internal/dashboard/interfaces/http"
)

func TestFormatBRL(t *testing.T) {
	assert.Equal(t, "R$ 0,00", dashhttp.FormatBRL(0))
	assert.Equal(t, "R$ 1,00", dashhttp.FormatBRL(100))
	assert.Equal(t, "R$ 12,34", dashhttp.FormatBRL(1234))
	assert.Equal(t, "R$ 1.234,56", dashhttp.FormatBRL(123456))
	assert.Equal(t, "R$ 1.234.567,89", dashhttp.FormatBRL(123456789))
	assert.Equal(t, "-R$ 12,34", dashhttp.FormatBRL(-1234))
}

func TestFormatHours(t *testing.T) {
	assert.Equal(t, "0.0h", dashhttp.FormatHours(0))
	assert.Equal(t, "1.5h", dashhttp.FormatHours(1.5))
	assert.Equal(t, "23.9h", dashhttp.FormatHours(23.9))
	assert.Equal(t, "1d", dashhttp.FormatHours(24))
	assert.Equal(t, "1d 1h", dashhttp.FormatHours(25))
	assert.Equal(t, "2d 5h", dashhttp.FormatHours(53))
}

func TestFormatPct(t *testing.T) {
	assert.Equal(t, "0%", dashhttp.FormatPct(0))
	assert.Equal(t, "50%", dashhttp.FormatPct(50))
	assert.Equal(t, "66.7%", dashhttp.FormatPct(66.66))
	assert.Equal(t, "100%", dashhttp.FormatPct(100))
}

func TestToTenantView_NilInputProducesEmpty(t *testing.T) {
	vm := dashhttp.ToTenantView(nil)
	assert.NotNil(t, vm)
	assert.Equal(t, int64(0), vm.Bloco1_Funil.Total)
}

func TestToTenantView_PopulatesAllBlocks(t *testing.T) {
	s := &domain.TenantStats{
		Bloco1_Funil: domain.FunilBlock{
			StatusTotals:  domain.LeadStatusCount{Open: 3, Won: 2, Lost: 1},
			ConversionPct: 66.66,
			NewToday:      1, NewThisWeek: 5,
			ColumnTotals: []domain.ColumnLeadsCount{
				{ColumnID: "c1", ColumnName: "Novos", OrderIndex: 0, Count: 3},
			},
		},
		Bloco2_WhatsApp: domain.WhatsAppStats{IncomingMessages: 10, OutgoingMessages: 8, ActiveConversations: 4, FirstResponseAvgSec: 90},
		Bloco3_Responsaveis: []domain.ResponsiblePerformance{
			{UserID: "u1", UserName: "Maria", Total: 5, Won: 2, Lost: 1},
		},
		Bloco4_TempoFunil: []domain.ColumnDwell{
			{ColumnID: "c1", ColumnName: "Novos", OrderIndex: 0, AvgHours: 12.5, StuckOver7Days: 1},
		},
		Bloco5_Produtos: []domain.ProductLeadsCount{
			{ProductID: "p1", ProductName: "Consulta", Total: 4, Won: 1, Lost: 0},
		},
		ScopeIsUser: true, CurrentUserName: "Maria", ActiveFunnelName: "Funil Comercial",
	}
	vm := dashhttp.ToTenantView(s)
	require.NotNil(t, vm)
	assert.Equal(t, int64(6), vm.Bloco1_Funil.Total) // 3+2+1
	assert.Equal(t, "66.7%", vm.Bloco1_Funil.ConversionPct)
	require.Len(t, vm.Bloco1_Funil.ColumnTotals, 1)
	assert.Equal(t, "1.5min", vm.Bloco2_WhatsApp.FirstResponseAvg) // 90s
	require.Len(t, vm.Bloco3_Responsaveis, 1)
	assert.Equal(t, "66.7%", vm.Bloco3_Responsaveis[0].ConversionPct) // 2/(2+1)
	require.Len(t, vm.Bloco4_TempoFunil, 1)
	assert.Equal(t, "12.5h", vm.Bloco4_TempoFunil[0].AvgHours)
	assert.True(t, vm.ScopeIsUser)
	assert.Equal(t, "Maria", vm.CurrentUserName)
	// JSON pré-serializado p/ Chart.js no template.
	assert.Contains(t, string(vm.Bloco1_Funil.ColumnTotalsJSON), `"ColumnName":"Novos"`)
	assert.Contains(t, string(vm.Bloco1_Funil.ColumnTotalsJSON), `"Count":3`)
	assert.Contains(t, string(vm.Bloco2_WhatsApp.MessagesJSON), `"in":10`)
	assert.Contains(t, string(vm.Bloco2_WhatsApp.MessagesJSON), `"out":8`)
}

func TestToAdminView_NilProducesEmpty(t *testing.T) {
	vm := dashhttp.ToAdminView(nil)
	assert.NotNil(t, vm)
	assert.Equal(t, int64(0), vm.Bloco1_Tenants.Total)
}

func TestToAdminView_PopulatesAllBlocks(t *testing.T) {
	s := &domain.AdminStats{
		Bloco1_Tenants: domain.TenantsBlock{
			Totals:       domain.TenantStatusCount{Active: 10, Inactive: 2, Blocked: 1},
			NewThisMonth: 3,
			Last6Months:  []domain.MonthlyCount{{Label: "2026-04", Count: 3}},
		},
		Bloco2_Uso: domain.UsageBlock{TotalLeads: 500, TotalMessages: 2000, ActiveConversations: 50},
		Bloco3_Health: domain.HealthBlock{
			Top10Active:  []domain.TenantActivity{{TenantID: "t1", TenantName: "T1", LeadCount: 5}},
			InactiveList: []domain.InactiveTenant{{TenantID: "t9", TenantName: "T9", DaysInactive: 60}},
		},
		Bloco4_Infra: domain.Infrastructure{
			APILatencyMs:   120.5,
			Error5xxRate:   0.001,
			ServicesStatus: []domain.ServiceStatus{{Name: "mysql", Up: true}},
		},
		Bloco5_Especialistas: domain.SpecialistsBlock{
			Total: 8, Qualifications: 0,
			ByTenant: []domain.SpecialistByTenant{{TenantID: "t1", TenantName: "T1", Total: 2}},
		},
		Bloco6_Financeiro: domain.FinancialBlock{
			ReceitaAnoCents:    1_500_000,
			PendenteTotalCents: 200_000,
			AtrasadoTotalCents: 50_000,
			PlanDist:           domain.PlanDistribution{Mensal: 5, Anual: 2, Vitalicio: 1, Externo: 0},
			TopOverdue:         []domain.OverdueTenant{{TenantID: "t9", TenantName: "T9", ValorCents: 50_000}},
		},
	}
	vm := dashhttp.ToAdminView(s)
	require.NotNil(t, vm)
	assert.Equal(t, int64(13), vm.Bloco1_Tenants.Total) // 10+2+1
	require.Len(t, vm.Bloco1_Tenants.Last6Months, 1)
	assert.Equal(t, "2026-04", vm.Bloco1_Tenants.Last6Months[0].Label)
	assert.Equal(t, int64(500), vm.Bloco2_Uso.TotalLeads)
	require.Len(t, vm.Bloco3_Health.Top10Active, 1)
	require.Len(t, vm.Bloco3_Health.InactiveList, 1)
	assert.Equal(t, "120.5ms", vm.Bloco4_Infra.APILatencyMs)
	assert.Equal(t, "0.1%", vm.Bloco4_Infra.Error5xxRate) // 0.001 * 100
	require.Len(t, vm.Bloco4_Infra.ServicesStatus, 1)
	assert.Equal(t, int64(8), vm.Bloco5_Especialistas.Total)
	assert.Equal(t, int64(8), vm.Bloco6_Financeiro.PlanDist.Total) // 5+2+1+0
	assert.Equal(t, "R$ 15.000,00", vm.Bloco6_Financeiro.ReceitaAno)
	require.Len(t, vm.Bloco6_Financeiro.TopOverdue, 1)
	assert.Equal(t, "R$ 500,00", vm.Bloco6_Financeiro.TopOverdue[0].Valor)
	assert.Equal(t, int64(50_000), vm.Bloco6_Financeiro.TopOverdue[0].ValorCents)
	// JSON pré-serializado p/ Chart.js no template.
	assert.Contains(t, string(vm.Bloco1_Tenants.MonthlyJSON), `"Label":"2026-04"`)
	assert.Contains(t, string(vm.Bloco3_Health.Top10JSON), `"TenantName":"T1"`)
	assert.Contains(t, string(vm.Bloco5_Especialistas.ByTenantJSON), `"TenantName":"T1"`)
	assert.Contains(t, string(vm.Bloco6_Financeiro.PlanDistJSON), `"Mensal":5`)
	assert.Contains(t, string(vm.Bloco6_Financeiro.TopOverdueJSON), `"ValorCents":50000`)
}
