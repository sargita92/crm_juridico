package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type GetUnreadTotalUseCase struct {
	conversationRepo domain.ConversationRepository
}

func NewGetUnreadTotalUseCase(conversationRepo domain.ConversationRepository) *GetUnreadTotalUseCase {
	return &GetUnreadTotalUseCase{conversationRepo: conversationRepo}
}

func (uc *GetUnreadTotalUseCase) Execute(ctx context.Context, tenantID string) (int, error) {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.get_unread_total",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	return uc.conversationRepo.SumUnreadByTenantID(ctx, tenantID)
}
