package application

import (
	"context"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type DeactivateTenantUseCase struct {
	repo      domain.TenantRepository
	publisher auditapp.Publisher
}

func NewDeactivateTenantUseCase(repo domain.TenantRepository) *DeactivateTenantUseCase {
	return &DeactivateTenantUseCase{repo: repo, publisher: auditapp.NoopPublisher{}}
}

// SetAuditPublisher injeta o publisher de auditoria. Quando nil, usa
// NoopPublisher (UC continua funcional sem audit em testes antigos).
func (uc *DeactivateTenantUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

func (uc *DeactivateTenantUseCase) Execute(ctx context.Context, id string) error {
	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := tenant.Deactivate(); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return err
	}

	uc.publishDeactivated(ctx, tenant)
	return nil
}

func (uc *DeactivateTenantUseCase) publishDeactivated(ctx context.Context, tenant *domain.Tenant) {
	actorEmail, actorID := actorFromContext(ctx)
	id := tenant.ID
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   &id,
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     auditdomain.ActionTenantDeactivated,
		Entity:     "tenant",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
	})
}
