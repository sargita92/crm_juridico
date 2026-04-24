package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type DisconnectWhatsAppUseCase struct {
	provider domain.WhatsAppProvider
}

func NewDisconnectWhatsAppUseCase(provider domain.WhatsAppProvider) *DisconnectWhatsAppUseCase {
	return &DisconnectWhatsAppUseCase{provider: provider}
}

func (uc *DisconnectWhatsAppUseCase) Execute(ctx context.Context, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.disconnect",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	return uc.provider.Disconnect(ctx, tenantID)
}
