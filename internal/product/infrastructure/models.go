package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

// --- Product model ---

type productModel struct {
	ID          string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID    string    `gorm:"column:tenant_id;type:char(36);not null"`
	Name        string    `gorm:"column:name;type:varchar(255);not null"`
	Description string    `gorm:"column:description;type:text"`
	Keywords    string    `gorm:"column:keywords;type:text"`
	Active      bool      `gorm:"column:active;type:tinyint(1);not null;default:1"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (productModel) TableName() string { return "products" }

func productToModel(p *domain.Product) *productModel {
	kw, _ := json.Marshal(p.Keywords)
	return &productModel{
		ID:          p.ID,
		TenantID:    p.TenantID,
		Name:        p.Name,
		Description: p.Description,
		Keywords:    string(kw),
		Active:      p.Active,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func productToDomain(m *productModel) *domain.Product {
	var keywords []string
	if m.Keywords != "" {
		_ = json.Unmarshal([]byte(m.Keywords), &keywords)
	}
	if keywords == nil {
		keywords = []string{}
	}
	return &domain.Product{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Name:        m.Name,
		Description: m.Description,
		Keywords:    keywords,
		Active:      m.Active,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// --- FunnelProduct model ---

type funnelProductModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	FunnelID  string    `gorm:"column:funnel_id;type:char(36);not null"`
	ProductID string    `gorm:"column:product_id;type:char(36);not null"`
	Priority  int       `gorm:"column:priority;type:int;not null;default:1"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (funnelProductModel) TableName() string { return "funnel_products" }

func funnelProductToModel(fp *domain.FunnelProduct) *funnelProductModel {
	return &funnelProductModel{
		ID:        fp.ID,
		FunnelID:  fp.FunnelID,
		ProductID: fp.ProductID,
		Priority:  fp.Priority,
		CreatedAt: fp.CreatedAt,
	}
}

func funnelProductToDomain(m *funnelProductModel) *domain.FunnelProduct {
	return &domain.FunnelProduct{
		ID:        m.ID,
		FunnelID:  m.FunnelID,
		ProductID: m.ProductID,
		Priority:  m.Priority,
		CreatedAt: m.CreatedAt,
	}
}

// --- PhoneNumber model ---

type phoneNumberModel struct {
	ID          string    `gorm:"primaryKey;column:id;type:char(36)"`
	ProductID   string    `gorm:"column:product_id;type:char(36);not null"`
	PhoneNumber string    `gorm:"column:phone_number;type:varchar(50);not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (phoneNumberModel) TableName() string { return "product_phone_numbers" }

func phoneNumberToModel(pn *domain.ProductPhoneNumber) *phoneNumberModel {
	return &phoneNumberModel{
		ID:          pn.ID,
		ProductID:   pn.ProductID,
		PhoneNumber: pn.PhoneNumber,
		CreatedAt:   pn.CreatedAt,
	}
}

func phoneNumberToDomain(m *phoneNumberModel) *domain.ProductPhoneNumber {
	return &domain.ProductPhoneNumber{
		ID:          m.ID,
		ProductID:   m.ProductID,
		PhoneNumber: m.PhoneNumber,
		CreatedAt:   m.CreatedAt,
	}
}
