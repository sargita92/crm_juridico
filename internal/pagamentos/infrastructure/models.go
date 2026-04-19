package infrastructure

import (
	"time"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type paymentModel struct {
	ID                string     `gorm:"column:id;primaryKey"`
	TenantID          string     `gorm:"column:tenant_id"`
	Tipo              string     `gorm:"column:tipo"`
	Descricao         *string    `gorm:"column:descricao"`
	ValorCents        int64      `gorm:"column:valor_cents"`
	Status            string     `gorm:"column:status"`
	Competencia       *time.Time `gorm:"column:competencia"`
	DataVencimento    time.Time  `gorm:"column:data_vencimento"`
	DataPagamento     *time.Time `gorm:"column:data_pagamento"`
	PaidByUserID      *string    `gorm:"column:paid_by_user_id"`
	CancelledAt       *time.Time `gorm:"column:cancelled_at"`
	CancelledByUserID *string    `gorm:"column:cancelled_by_user_id"`
	Observacao        *string    `gorm:"column:observacao"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (paymentModel) TableName() string { return "payments" }

func toModel(p *domain.Payment) *paymentModel {
	var desc *string
	if p.Descricao != "" {
		d := p.Descricao
		desc = &d
	}
	var obs *string
	if p.Observacao != "" {
		o := p.Observacao
		obs = &o
	}
	return &paymentModel{
		ID:                p.ID,
		TenantID:          p.TenantID,
		Tipo:              string(p.Tipo),
		Descricao:         desc,
		ValorCents:        p.ValorCents,
		Status:            string(p.Status),
		Competencia:       p.Competencia,
		DataVencimento:    p.DataVencimento,
		DataPagamento:     p.DataPagamento,
		PaidByUserID:      p.PaidByUserID,
		CancelledAt:       p.CancelledAt,
		CancelledByUserID: p.CancelledByUserID,
		Observacao:        obs,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func toDomain(m *paymentModel) *domain.Payment {
	desc := ""
	if m.Descricao != nil {
		desc = *m.Descricao
	}
	obs := ""
	if m.Observacao != nil {
		obs = *m.Observacao
	}
	return &domain.Payment{
		ID:                m.ID,
		TenantID:          m.TenantID,
		Tipo:              domain.PaymentType(m.Tipo),
		Descricao:         desc,
		ValorCents:        m.ValorCents,
		Status:            domain.PaymentStatus(m.Status),
		Competencia:       m.Competencia,
		DataVencimento:    m.DataVencimento,
		DataPagamento:     m.DataPagamento,
		PaidByUserID:      m.PaidByUserID,
		CancelledAt:       m.CancelledAt,
		CancelledByUserID: m.CancelledByUserID,
		Observacao:        obs,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}
