package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestNewBillingConfig_Mensal_Success(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	valor := int64(50000)
	dia := uint8(10)
	cfg, err := domain.NewBillingConfig(domain.PlanMensal, &valor, &dia, &start, true)
	require.NoError(t, err)
	assert.True(t, cfg.GenerateRecurring())
	assert.True(t, cfg.ShowsPortalMenu())
}

func TestNewBillingConfig_Anual_Success(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	valor := int64(600000)
	dia := uint8(15)
	cfg, err := domain.NewBillingConfig(domain.PlanAnual, &valor, &dia, &start, true)
	require.NoError(t, err)
	assert.True(t, cfg.GenerateRecurring())
}

func TestNewBillingConfig_Vitalicio_IgnoresFields(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	valor := int64(1000)
	dia := uint8(10)
	cfg, err := domain.NewBillingConfig(domain.PlanVitalicio, &valor, &dia, &start, true)
	require.NoError(t, err)
	assert.False(t, cfg.GenerateRecurring())
	assert.False(t, cfg.ShowsPortalMenu())
	assert.Nil(t, cfg.ValorCents)
	assert.Nil(t, cfg.DiaVencimento)
	assert.Nil(t, cfg.DataInicioCobranca)
}

func TestNewBillingConfig_Externo_IgnoresFields(t *testing.T) {
	cfg, err := domain.NewBillingConfig(domain.PlanExterno, nil, nil, nil, true)
	require.NoError(t, err)
	assert.False(t, cfg.GenerateRecurring())
	assert.False(t, cfg.ShowsPortalMenu())
}

func TestNewBillingConfig_InvalidPlano(t *testing.T) {
	_, err := domain.NewBillingConfig(domain.Plan("bogus"), nil, nil, nil, true)
	assert.ErrorIs(t, err, domain.ErrInvalidPlano)
}

func TestNewBillingConfig_Mensal_RequiresValor(t *testing.T) {
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err := domain.NewBillingConfig(domain.PlanMensal, nil, &dia, &start, true)
	assert.ErrorIs(t, err, domain.ErrValorInvalido)
}

func TestNewBillingConfig_Mensal_RejectsZeroValor(t *testing.T) {
	valor := int64(0)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err := domain.NewBillingConfig(domain.PlanMensal, &valor, &dia, &start, true)
	assert.ErrorIs(t, err, domain.ErrValorInvalido)
}

func TestNewBillingConfig_Mensal_RequiresDia(t *testing.T) {
	valor := int64(1000)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	_, err := domain.NewBillingConfig(domain.PlanMensal, &valor, nil, &start, true)
	assert.ErrorIs(t, err, domain.ErrInvalidPlano)
}

func TestNewBillingConfig_Mensal_RejectsDiaForaIntervalo(t *testing.T) {
	valor := int64(1000)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	cases := []uint8{0, 29, 30, 31, 200}
	for _, dia := range cases {
		d := dia
		_, err := domain.NewBillingConfig(domain.PlanMensal, &valor, &d, &start, true)
		assert.ErrorIsf(t, err, domain.ErrInvalidPlano, "dia=%d deveria ser invalido", d)
	}
}

func TestNewBillingConfig_Mensal_RequiresDataInicio(t *testing.T) {
	valor := int64(1000)
	dia := uint8(10)
	_, err := domain.NewBillingConfig(domain.PlanMensal, &valor, &dia, nil, true)
	assert.ErrorIs(t, err, domain.ErrInvalidPlano)
}

func TestShowsPortalMenu_RequiresExibirFlag(t *testing.T) {
	valor := int64(1000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	cfg, _ := domain.NewBillingConfig(domain.PlanMensal, &valor, &dia, &start, false)
	assert.True(t, cfg.GenerateRecurring())
	assert.False(t, cfg.ShowsPortalMenu())
}
