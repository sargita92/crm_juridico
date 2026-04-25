package application

import (
	"context"
	"time"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	pagdomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
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
	repo      domain.TenantRepository
	publisher auditapp.Publisher
}

func NewUpdateTenantUseCase(repo domain.TenantRepository) *UpdateTenantUseCase {
	return &UpdateTenantUseCase{repo: repo, publisher: auditapp.NoopPublisher{}}
}

// SetAuditPublisher injeta o publisher de auditoria. Quando nil, mantem o
// NoopPublisher default. Decisao F12: update sem mudancas (diff vazio) NAO
// publica, para evitar ruido na timeline de auditoria.
func (uc *UpdateTenantUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

func (uc *UpdateTenantUseCase) Execute(ctx context.Context, input UpdateTenantInput) (*GetTenantOutput, error) {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	before := snapshotTenant(tenant)

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

	uc.publishUpdated(ctx, tenant, before)

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

func (uc *UpdateTenantUseCase) publishUpdated(ctx context.Context, tenant *domain.Tenant, before map[string]any) {
	after := snapshotTenant(tenant)
	diff := auditapp.BuildDiff(before, after)
	if diff == nil {
		// Sem mudancas auditaveis — politica do projeto e nao publicar
		// para evitar entries duplicadas na timeline (justifica spec
		// Step 6: "se diff == nil ainda publica? NAO").
		return
	}
	actorEmail, actorID := actorFromContext(ctx)
	id := tenant.ID
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   &id,
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     auditdomain.ActionTenantUpdated,
		Entity:     "tenant",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
		Metadata:   auditdomain.Metadata{"diff": diff},
	})
}
