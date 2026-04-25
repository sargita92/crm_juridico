package application

import (
	"context"

	"github.com/google/uuid"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type CreateTenantInput struct {
	Name     string
	Type     string
	Document string
}

type CreateTenantOutput struct {
	ID       string
	Name     string
	Type     string
	Document string
	Status   string
}

type CreateTenantUseCase struct {
	repo      domain.TenantRepository
	publisher auditapp.Publisher
}

func NewCreateTenantUseCase(repo domain.TenantRepository) *CreateTenantUseCase {
	return &CreateTenantUseCase{repo: repo, publisher: auditapp.NoopPublisher{}}
}

// SetAuditPublisher injeta o publisher de auditoria. Quando nil, mantem o
// NoopPublisher default — UC continua funcionando sem audit em testes
// antigos. Erros do publisher sao engolidos pela politica F12 (auditoria
// nao quebra a operacao).
func (uc *CreateTenantUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

func (uc *CreateTenantUseCase) Execute(ctx context.Context, input CreateTenantInput) (*CreateTenantOutput, error) {
	tenant, err := domain.NewTenant(uuid.New().String(), input.Name, domain.TenantType(input.Type), input.Document)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	uc.publishCreated(ctx, tenant)

	return &CreateTenantOutput{
		ID:       tenant.ID,
		Name:     tenant.Name,
		Type:     string(tenant.Type),
		Document: tenant.Document,
		Status:   string(tenant.Status),
	}, nil
}

func (uc *CreateTenantUseCase) publishCreated(ctx context.Context, tenant *domain.Tenant) {
	actorEmail, actorID := actorFromContext(ctx)
	id := tenant.ID
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   &id,
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     auditdomain.ActionTenantCreated,
		Entity:     "tenant",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
		Metadata: auditdomain.Metadata{
			"name": tenant.Name,
			"type": string(tenant.Type),
		},
	})
}
