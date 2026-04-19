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
}
