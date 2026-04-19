package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestToPaymentView_AvulsoPendente(t *testing.T) {
	p, err := domain.NewAvulsoPayment("p1", "t1", "Setup", 15000,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "obs")
	require.NoError(t, err)
	v := toPaymentView(*p)
	assert.Equal(t, "p1", v.ID)
	assert.Equal(t, "avulso", v.Tipo)
	assert.Equal(t, "R$ 150.00", v.Valor)
	assert.Equal(t, "pendente", v.Status)
	assert.Equal(t, "10/05/2026", v.DataVencimento)
	assert.Empty(t, v.Competencia)
	assert.True(t, v.PodeMarcarPago)
	assert.True(t, v.PodeCancelar)
}

func TestToPaymentView_RecorrentePago(t *testing.T) {
	comp := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	p, err := domain.NewRecorrentePayment("p1", "t1", 50000, comp, time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	_ = p.MarkAsPaid("u1", time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC))
	v := toPaymentView(*p)
	assert.Equal(t, "pago", v.Status)
	assert.Equal(t, "04/2026", v.Competencia)
	assert.Equal(t, "05/04/2026 12:00", v.DataPagamento)
	assert.False(t, v.PodeMarcarPago)
	assert.False(t, v.PodeCancelar)
}

func TestToSummaryView_EmDia(t *testing.T) {
	s := &application.FinancialSummaryOutput{
		Badge:             application.BadgeEmDia,
		TotalPagoAnoCents: 120000,
	}
	v := toSummaryView(s)
	assert.Equal(t, "em_dia", v.Badge)
	assert.Equal(t, "Em dia", v.BadgeText)
	assert.Equal(t, "R$ 1200.00", v.TotalPagoAno)
}

func TestParseValorCents(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"50.00", 5000, false},
		{"50,00", 5000, false},
		{"0.01", 1, false},
		{"", 0, true},
		{"0", 0, true},
		{"abc", 0, true},
	}
	for _, c := range cases {
		got, err := parseValorCents(c.in)
		if c.wantErr {
			assert.Error(t, err, c.in)
			continue
		}
		require.NoError(t, err, c.in)
		assert.Equal(t, c.want, got, c.in)
	}
}

type stubQuery map[string]string

func (s stubQuery) Query(k string) string { return s[k] }

func TestParseFiltersFromQuery(t *testing.T) {
	q := stubQuery{
		"status":       "pendente",
		"data_inicial": "2026-04-01",
		"data_final":   "2026-04-30",
		"page":         "2",
	}
	status, di, df, page := parseFiltersFromQuery(q)
	require.NotNil(t, status)
	assert.Equal(t, domain.StatusPendente, *status)
	require.NotNil(t, di)
	assert.Equal(t, 2026, di.Year())
	require.NotNil(t, df)
	assert.Equal(t, 30, df.Day())
	assert.Equal(t, 2, page)
}

func TestParseFiltersFromQuery_DefaultsPage1(t *testing.T) {
	_, _, _, page := parseFiltersFromQuery(stubQuery{})
	assert.Equal(t, 1, page)
}

func TestStatusBadgeClass_TodosOsStatus(t *testing.T) {
	cases := []struct {
		s    domain.PaymentStatus
		want string
	}{
		{domain.StatusPago, "badge badge-success"},
		{domain.StatusPendente, "badge badge-warning"},
		{domain.StatusAtrasado, "badge badge-danger"},
		{domain.StatusCancelado, "badge badge-muted"},
		{"desconhecido", "badge"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, statusBadgeClass(c.s), string(c.s))
	}
}

func TestParseDate_InvalidRetornaErro(t *testing.T) {
	_, err := parseDate("nao-e-data")
	assert.Error(t, err)
}
