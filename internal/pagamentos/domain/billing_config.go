package domain

import "time"

// BillingConfig é a configuração de cobrança que vive no agregado tenant
// e governa o comportamento do módulo de pagamentos.
type BillingConfig struct {
	Plano              Plan
	ValorCents         *int64
	DiaVencimento      *uint8
	DataInicioCobranca *time.Time
	ExibirPagamentos   bool
}

// NewBillingConfig valida e constrói uma BillingConfig. Para planos mensal/annual,
// valor, dia (1..28) e data de início são obrigatórios. Para vitalicio/externo
// esses campos são ignorados (zerados).
func NewBillingConfig(plano Plan, valor *int64, dia *uint8, start *time.Time, exibir bool) (*BillingConfig, error) {
	if !IsValidPlano(plano) {
		return nil, ErrInvalidPlano
	}
	if plano == PlanMensal || plano == PlanAnual {
		if valor == nil || *valor <= 0 {
			return nil, ErrValorInvalido
		}
		if dia == nil || *dia < 1 || *dia > 28 {
			return nil, ErrInvalidPlano
		}
		if start == nil {
			return nil, ErrInvalidPlano
		}
	} else {
		valor = nil
		dia = nil
		start = nil
	}
	return &BillingConfig{
		Plano:              plano,
		ValorCents:         valor,
		DiaVencimento:      dia,
		DataInicioCobranca: start,
		ExibirPagamentos:   exibir,
	}, nil
}

// GenerateRecurring indica se este tenant gera lançamentos recorrentes via cron.
func (c *BillingConfig) GenerateRecurring() bool {
	return c.Plano == PlanMensal || c.Plano == PlanAnual
}

// ShowsPortalMenu indica se o tenant deve ver o item "Pagamentos" no portal.
// Vitalício e externo nunca veem; mensal/annual veem apenas se a flag estiver ligada.
func (c *BillingConfig) ShowsPortalMenu() bool {
	return c.GenerateRecurring() && c.ExibirPagamentos
}
