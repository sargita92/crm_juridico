package domain

import (
	"errors"
	"time"
)

type TenantProduct struct {
	ID        string
	TenantID  string
	ProductID string
	CreatedAt time.Time
}

func NewTenantProduct(id, tenantID, productID string) (*TenantProduct, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if productID == "" {
		return nil, errors.New("product ID is required")
	}
	return &TenantProduct{
		ID: id, TenantID: tenantID, ProductID: productID,
		CreatedAt: time.Now(),
	}, nil
}
