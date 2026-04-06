package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ConnectWhatsAppUseCase struct {
	provider domain.WhatsAppProvider
}

func NewConnectWhatsAppUseCase(provider domain.WhatsAppProvider) *ConnectWhatsAppUseCase {
	return &ConnectWhatsAppUseCase{provider: provider}
}

func (uc *ConnectWhatsAppUseCase) Execute(ctx context.Context, tenantID string) (<-chan string, error) {
	return uc.provider.Connect(ctx, tenantID)
}
