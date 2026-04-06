package domain

import "time"

const MaxContactNameLength = 255

type Contact struct {
	ID         string
	TenantID   string
	Name       string
	Phone      string
	WhatsAppID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewContact(id, tenantID, name, phone, whatsappID string) (*Contact, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if phone == "" {
		return nil, ErrContactPhoneRequired
	}
	if whatsappID == "" {
		return nil, ErrContactWhatsAppIDRequired
	}
	if len(name) > MaxContactNameLength {
		return nil, ErrContactNameTooLong
	}
	now := time.Now()
	return &Contact{
		ID:         id,
		TenantID:   tenantID,
		Name:       name,
		Phone:      phone,
		WhatsAppID: whatsappID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (c *Contact) UpdateName(name string) {
	if name != "" && name != c.Name {
		c.Name = name
		c.UpdatedAt = time.Now()
	}
}
