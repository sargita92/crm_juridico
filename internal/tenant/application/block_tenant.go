package application

import (
	"context"

	"github.com/google/uuid"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type BlockTenantInput struct {
	ID          string
	Reason      string
	PerformedBy string
}

type BlockTenantUseCase struct {
	repo        domain.TenantRepository
	historyRepo domain.BlockHistoryRepository
	publisher   auditapp.Publisher
}

func NewBlockTenantUseCase(repo domain.TenantRepository, historyRepo domain.BlockHistoryRepository) *BlockTenantUseCase {
	return &BlockTenantUseCase{repo: repo, historyRepo: historyRepo, publisher: auditapp.NoopPublisher{}}
}

// SetAuditPublisher injeta o publisher de auditoria. Quando nil, usa
// NoopPublisher (UC continua funcional sem audit em testes antigos).
func (uc *BlockTenantUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

func (uc *BlockTenantUseCase) Execute(ctx context.Context, input BlockTenantInput) error {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := tenant.Block(input.Reason); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return err
	}

	entry, err := domain.NewBlockHistoryEntry(uuid.New().String(), input.ID, domain.BlockActionBlock, input.Reason, input.PerformedBy)
	if err != nil {
		return err
	}

	if err := uc.historyRepo.Save(ctx, entry); err != nil {
		return err
	}

	uc.publishBlocked(ctx, input)
	return nil
}

func (uc *BlockTenantUseCase) publishBlocked(ctx context.Context, input BlockTenantInput) {
	actorEmail, actorID := actorFromContext(ctx)
	id := input.ID
	// PerformedBy carrega o id do operador admin diretamente do handler;
	// quando presente, usamos como UserID (alinhado a user_id no log).
	if actorID == nil && input.PerformedBy != "" {
		pb := input.PerformedBy
		actorID = &pb
	}
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   &id,
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     auditdomain.ActionTenantBlocked,
		Entity:     "tenant",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
		Metadata:   auditdomain.Metadata{"reason": input.Reason},
	})
}
