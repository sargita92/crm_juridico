package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestNewAvulsoPayment_Success(t *testing.T) {
	p, err := domain.NewAvulsoPayment("id-1", "tenant-1", "Setup inicial", 15000,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "obs")
	require.NoError(t, err)
	assert.Equal(t, domain.TypeAvulso, p.Tipo)
	assert.Equal(t, domain.StatusPendente, p.Status)
	assert.Equal(t, int64(15000), p.ValorCents)
	assert.Nil(t, p.Competencia)
}

func TestNewAvulsoPayment_RequiresDescricao(t *testing.T) {
	_, err := domain.NewAvulsoPayment("id-1", "tenant-1", "", 15000,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "")
	assert.ErrorIs(t, err, domain.ErrDescricaoRequired)
}

func TestNewAvulsoPayment_RejectsZeroValor(t *testing.T) {
	_, err := domain.NewAvulsoPayment("id-1", "tenant-1", "Setup", 0,
		time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), "")
	assert.ErrorIs(t, err, domain.ErrValorInvalido)
}

func TestNewAvulsoPayment_RequiresTenantID(t *testing.T) {
	_, err := domain.NewAvulsoPayment("id-1", "", "Setup", 1000, time.Now(), "")
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestNewAvulsoPayment_RequiresVencimento(t *testing.T) {
	_, err := domain.NewAvulsoPayment("id-1", "tenant-1", "Setup", 1000, time.Time{}, "")
	assert.ErrorIs(t, err, domain.ErrDataVencimentoRequired)
}

func TestNewRecorrentePayment_Success(t *testing.T) {
	comp := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	venc := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	p, err := domain.NewRecorrentePayment("id-2", "tenant-1", 50000, comp, venc)
	require.NoError(t, err)
	assert.Equal(t, domain.TypeRecorrente, p.Tipo)
	assert.Equal(t, domain.StatusPendente, p.Status)
	require.NotNil(t, p.Competencia)
	assert.Equal(t, comp, *p.Competencia)
	assert.Equal(t, venc, p.DataVencimento)
	assert.Empty(t, p.Descricao)
}

func TestNewRecorrentePayment_RejectsZeroValor(t *testing.T) {
	comp := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err := domain.NewRecorrentePayment("id-2", "tenant-1", 0, comp, comp.AddDate(0, 0, 10))
	assert.ErrorIs(t, err, domain.ErrValorInvalido)
}

func TestMarkAsPaid_FromPendente(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	when := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	err := p.MarkAsPaid("user-1", when)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPago, p.Status)
	require.NotNil(t, p.PaidByUserID)
	assert.Equal(t, "user-1", *p.PaidByUserID)
	require.NotNil(t, p.DataPagamento)
	assert.Equal(t, when, *p.DataPagamento)
}

func TestMarkAsPaid_FromAtrasado(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	p.Status = domain.StatusAtrasado
	err := p.MarkAsPaid("user-1", time.Now())
	require.NoError(t, err)
	assert.Equal(t, domain.StatusPago, p.Status)
}

func TestMarkAsPaid_FromPago_Rejects(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	p.Status = domain.StatusPago
	err := p.MarkAsPaid("user-1", time.Now())
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestMarkAsPaid_FromCancelado_Rejects(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	p.Status = domain.StatusCancelado
	err := p.MarkAsPaid("user-1", time.Now())
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestCancel_RequiresMotivo(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	err := p.Cancel("user-1", "")
	assert.ErrorIs(t, err, domain.ErrMotivoRequired)
}

func TestCancel_FromPago_Rejects(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	p.Status = domain.StatusPago
	err := p.Cancel("user-1", "Erro de lancamento")
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestCancel_FromCancelado_Rejects(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	p.Status = domain.StatusCancelado
	err := p.Cancel("user-1", "x")
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestCancel_FromPendente_Success(t *testing.T) {
	p, _ := domain.NewAvulsoPayment("id", "t", "d", 1000, time.Now(), "")
	err := p.Cancel("user-1", "Duplicado")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusCancelado, p.Status)
	assert.Equal(t, "Duplicado", p.Observacao)
	require.NotNil(t, p.CancelledByUserID)
	assert.Equal(t, "user-1", *p.CancelledByUserID)
	require.NotNil(t, p.CancelledAt)
}

func TestIsValidPlano(t *testing.T) {
	assert.True(t, domain.IsValidPlano(domain.PlanMensal))
	assert.True(t, domain.IsValidPlano(domain.PlanAnual))
	assert.True(t, domain.IsValidPlano(domain.PlanVitalicio))
	assert.True(t, domain.IsValidPlano(domain.PlanExterno))
	assert.False(t, domain.IsValidPlano(domain.Plan("bogus")))
}
