package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type DisconnectWhatsAppUseCase struct {
	provider domain.WhatsAppProvider
}

func NewDisconnectWhatsAppUseCase(provider domain.WhatsAppProvider) *DisconnectWhatsAppUseCase {
	return &DisconnectWhatsAppUseCase{provider: provider}
}

func (uc *DisconnectWhatsAppUseCase) Execute(ctx context.Context, tenantID string) error {
	return uc.provider.Disconnect(ctx, tenantID)
}
