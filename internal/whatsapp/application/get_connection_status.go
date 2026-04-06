package application

import (
	"context"

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

func (uc *GetConnectionStatusUseCase) Execute(_ context.Context, tenantID string) (*ConnectionStatus, error) {
	return &ConnectionStatus{
		Connected: uc.provider.IsConnected(tenantID),
	}, nil
}
