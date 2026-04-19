package infrastructure

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

// tenantBillingRow reflete apenas as colunas da tabela tenants que o módulo
// pagamentos precisa. Não importa o pacote tenant para evitar acoplamento.
type tenantBillingRow struct {
	ID                 string     `gorm:"column:id"`
	Status             string     `gorm:"column:status"`
	Plano              string     `gorm:"column:plano"`
	ValorCobrancaCents *int64     `gorm:"column:valor_cobranca_cents"`
	DiaVencimento      *uint8     `gorm:"column:dia_vencimento"`
	DataInicioCobranca *time.Time `gorm:"column:data_inicio_cobranca"`
	ExibirPagamentos   bool       `gorm:"column:exibir_pagamentos"`
}

// GormTenantBillingRepository lê configuração de cobrança direto da tabela
// tenants via Gorm, traduzindo para o modelo de domínio do pagamentos.
type GormTenantBillingRepository struct {
	db *gorm.DB
}

func NewGormTenantBillingRepository(db *gorm.DB) *GormTenantBillingRepository {
	return &GormTenantBillingRepository{db: db}
}

func (r *GormTenantBillingRepository) GetByID(ctx context.Context, tenantID string) (*domain.TenantBilling, error) {
	var row tenantBillingRow
	err := r.db.WithContext(ctx).
		Table("tenants").
		Select("id, status, plano, valor_cobranca_cents, dia_vencimento, data_inicio_cobranca, exibir_pagamentos").
		Where("id = ?", tenantID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, err
	}
	cfg, err := domain.NewBillingConfig(
		domain.Plan(row.Plano),
		row.ValorCobrancaCents,
		row.DiaVencimento,
		row.DataInicioCobranca,
		row.ExibirPagamentos,
	)
	if err != nil {
		return nil, err
	}
	return &domain.TenantBilling{
		TenantID: row.ID,
		Active:   row.Status == "active",
		Config:   *cfg,
	}, nil
}

func (r *GormTenantBillingRepository) ListActiveBillable(ctx context.Context) ([]domain.TenantBilling, error) {
	var rows []tenantBillingRow
	err := r.db.WithContext(ctx).
		Table("tenants").
		Select("id, status, plano, valor_cobranca_cents, dia_vencimento, data_inicio_cobranca, exibir_pagamentos").
		Where("status = ? AND plano IN (?, ?)", "active", string(domain.PlanMensal), string(domain.PlanAnual)).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]domain.TenantBilling, 0, len(rows))
	for _, row := range rows {
		cfg, err := domain.NewBillingConfig(
			domain.Plan(row.Plano),
			row.ValorCobrancaCents,
			row.DiaVencimento,
			row.DataInicioCobranca,
			row.ExibirPagamentos,
		)
		if err != nil {
			// Tenant mal configurado: skip sem falhar o batch inteiro.
			continue
		}
		out = append(out, domain.TenantBilling{
			TenantID: row.ID,
			Active:   true,
			Config:   *cfg,
		})
	}
	return out, nil
}
