package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type WhatsAppContactAdapter struct {
	contactRepo whatsappdomain.ContactRepository
}

func NewWhatsAppContactAdapter(contactRepo whatsappdomain.ContactRepository) *WhatsAppContactAdapter {
	return &WhatsAppContactAdapter{contactRepo: contactRepo}
}

func (a *WhatsAppContactAdapter) FindByID(ctx context.Context, contactID string) (funneldomain.ContactInfo, error) {
	contact, err := a.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return funneldomain.ContactInfo{}, err
	}
	return funneldomain.ContactInfo{
		Name:  contact.Name,
		Phone: contact.Phone,
	}, nil
}
