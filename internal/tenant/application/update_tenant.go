package application

import (
	"context"
	"time"

	pagdomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type UpdateTenantBilling struct {
	Plano              string
	ValorCobrancaCents *int64
	DiaVencimento      *uint8
	DataInicioCobranca *time.Time
	ExibirPagamentos   bool
}

type UpdateTenantInput struct {
	ID       string
	Name     string
	Type     string
	Document string
	Billing  *UpdateTenantBilling
}

type UpdateTenantUseCase struct {
	repo domain.TenantRepository
}

func NewUpdateTenantUseCase(repo domain.TenantRepository) *UpdateTenantUseCase {
	return &UpdateTenantUseCase{repo: repo}
}

func (uc *UpdateTenantUseCase) Execute(ctx context.Context, input UpdateTenantInput) (*GetTenantOutput, error) {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := tenant.Update(input.Name, domain.TenantType(input.Type), input.Document); err != nil {
		return nil, err
	}

	if input.Billing != nil {
		cfg, err := pagdomain.NewBillingConfig(
			pagdomain.Plan(input.Billing.Plano),
			input.Billing.ValorCobrancaCents,
			input.Billing.DiaVencimento,
			input.Billing.DataInicioCobranca,
			input.Billing.ExibirPagamentos,
		)
		if err != nil {
			return nil, err
		}
		tenant.SetBillingConfig(string(cfg.Plano), cfg.ValorCents, cfg.DiaVencimento, cfg.DataInicioCobranca, cfg.ExibirPagamentos)
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return &GetTenantOutput{
		ID:                 tenant.ID,
		Name:               tenant.Name,
		Type:               string(tenant.Type),
		Document:           tenant.Document,
		Status:             string(tenant.Status),
		BlockReason:        tenant.BlockReason,
		Plano:              tenant.Plano,
		ValorCobrancaCents: tenant.ValorCobrancaCents,
		DiaVencimento:      tenant.DiaVencimento,
		DataInicioCobranca: tenant.DataInicioCobranca,
		ExibirPagamentos:   tenant.ExibirPagamentos,
		CreatedAt:          tenant.CreatedAt,
		UpdatedAt:          tenant.UpdatedAt,
	}, nil
}
