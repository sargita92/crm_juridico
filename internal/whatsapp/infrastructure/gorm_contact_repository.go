package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type GormContactRepository struct {
	db *gorm.DB
}

func NewGormContactRepository(db *gorm.DB) *GormContactRepository {
	return &GormContactRepository{db: db}
}

func (r *GormContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	model := contactToModel(contact)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormContactRepository) FindByID(ctx context.Context, id string) (*domain.Contact, error) {
	var model contactModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrContactNotFound
		}
		return nil, err
	}
	return contactToDomain(&model), nil
}

func (r *GormContactRepository) FindByWhatsAppID(ctx context.Context, tenantID, whatsappID string) (*domain.Contact, error) {
	var model contactModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND whatsapp_id = ?", tenantID, whatsappID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrContactNotFound
		}
		return nil, err
	}
	return contactToDomain(&model), nil
}

func (r *GormContactRepository) Update(ctx context.Context, contact *domain.Contact) error {
	model := contactToModel(contact)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrContactNotFound
	}
	return nil
}

func contactToModel(c *domain.Contact) *contactModel {
	return &contactModel{
		ID:         c.ID,
		TenantID:   c.TenantID,
		Name:       c.Name,
		Phone:      c.Phone,
		WhatsAppID: c.WhatsAppID,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

func contactToDomain(m *contactModel) *domain.Contact {
	return &domain.Contact{
		ID:         m.ID,
		TenantID:   m.TenantID,
		Name:       m.Name,
		Phone:      m.Phone,
		WhatsAppID: m.WhatsAppID,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}
