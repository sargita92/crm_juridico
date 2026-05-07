package domain

import "time"

// Plan representa o plano comercial contratado pelo tenant.
type Plan string

const (
	PlanMensal    Plan = "mensal"
	PlanAnual     Plan = "annual"
	PlanVitalicio Plan = "vitalicio"
	PlanExterno   Plan = "externo"
)

// IsValidPlano retorna true se o plano está no conjunto de planos suportados.
func IsValidPlano(p Plan) bool {
	switch p {
	case PlanMensal, PlanAnual, PlanVitalicio, PlanExterno:
		return true
	default:
		return false
	}
}

// PaymentType distingue cobranças recorrentes (assinaturas) de lançamentos avulsos.
type PaymentType string

const (
	TypeRecorrente PaymentType = "recorrente"
	TypeAvulso     PaymentType = "avulso"
)

// PaymentStatus é o estado atual do pagamento.
type PaymentStatus string

const (
	StatusPendente  PaymentStatus = "pendente"
	StatusPago      PaymentStatus = "pago"
	StatusAtrasado  PaymentStatus = "atrasado"
	StatusCancelado PaymentStatus = "cancelado"
)

// Payment é a entidade de domínio que representa um lançamento financeiro
// associado a um tenant, podendo ser uma mensalidade recorrente ou um
// lançamento avulso.
type Payment struct {
	ID                string
	TenantID          string
	Tipo              PaymentType
	Descricao         string
	ValorCents        int64
	Status            PaymentStatus
	Competencia       *time.Time
	DataVencimento    time.Time
	DataPagamento     *time.Time
	PaidByUserID      *string
	CancelledAt       *time.Time
	CancelledByUserID *string
	Observacao        string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NewAvulsoPayment cria um lançamento avulso em estado pendente.
func NewAvulsoPayment(id, tenantID, descricao string, valorCents int64, vencimento time.Time, observacao string) (*Payment, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if descricao == "" {
		return nil, ErrDescricaoRequired
	}
	if valorCents <= 0 {
		return nil, ErrValorInvalido
	}
	if vencimento.IsZero() {
		return nil, ErrDataVencimentoRequired
	}
	now := time.Now()
	return &Payment{
		ID:             id,
		TenantID:       tenantID,
		Tipo:           TypeAvulso,
		Descricao:      descricao,
		ValorCents:     valorCents,
		Status:         StatusPendente,
		Competencia:    nil,
		DataVencimento: vencimento,
		Observacao:     observacao,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// NewRecorrentePayment cria um lançamento recorrente (mensalidade) em estado pendente.
func NewRecorrentePayment(id, tenantID string, valorCents int64, competencia, vencimento time.Time) (*Payment, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if valorCents <= 0 {
		return nil, ErrValorInvalido
	}
	if vencimento.IsZero() {
		return nil, ErrDataVencimentoRequired
	}
	now := time.Now()
	comp := competencia
	return &Payment{
		ID:             id,
		TenantID:       tenantID,
		Tipo:           TypeRecorrente,
		ValorCents:     valorCents,
		Status:         StatusPendente,
		Competencia:    &comp,
		DataVencimento: vencimento,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MarkAsPaid marca o pagamento como pago. Só é permitido transitar a partir
// de StatusPendente ou StatusAtrasado.
func (p *Payment) MarkAsPaid(userID string, when time.Time) error {
	if p.Status != StatusPendente && p.Status != StatusAtrasado {
		return ErrInvalidTransition
	}
	paidAt := when
	uid := userID
	p.Status = StatusPago
	p.PaidByUserID = &uid
	p.DataPagamento = &paidAt
	p.UpdatedAt = time.Now()
	return nil
}

// Cancel cancela o pagamento registrando o motivo. Não é permitido cancelar
// pagamentos já pagos ou já cancelados, e o motivo é obrigatório.
func (p *Payment) Cancel(userID, motivo string) error {
	if motivo == "" {
		return ErrMotivoRequired
	}
	if p.Status == StatusPago || p.Status == StatusCancelado {
		return ErrInvalidTransition
	}
	now := time.Now()
	uid := userID
	p.Status = StatusCancelado
	p.CancelledByUserID = &uid
	p.CancelledAt = &now
	p.Observacao = motivo
	p.UpdatedAt = now
	return nil
}
