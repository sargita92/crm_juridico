package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ConnectWhatsAppUseCase struct {
	provider domain.WhatsAppProvider
}

func NewConnectWhatsAppUseCase(provider domain.WhatsAppProvider) *ConnectWhatsAppUseCase {
	return &ConnectWhatsAppUseCase{provider: provider}
}

func (uc *ConnectWhatsAppUseCase) Execute(ctx context.Context, tenantID string) (<-chan string, error) {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.connect",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	return uc.provider.Connect(ctx, tenantID)
}
