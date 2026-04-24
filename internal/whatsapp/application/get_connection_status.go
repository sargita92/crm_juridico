package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ConnectionStatus struct {
	Connected bool
}

type GetConnectionStatusUseCase struct {
	provider domain.WhatsAppProvider
}

func NewGetConnectionStatusUseCase(provider domain.WhatsAppProvider) *GetConnectionStatusUseCase {
	return &GetConnectionStatusUseCase{provider: provider}
}

func (uc *GetConnectionStatusUseCase) Execute(ctx context.Context, tenantID string) (*ConnectionStatus, error) {
	_, span := observability.StartSpan(ctx, "whatsapp.usecase.get_connection_status",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	return &ConnectionStatus{
		Connected: uc.provider.IsConnected(tenantID),
	}, nil
}
