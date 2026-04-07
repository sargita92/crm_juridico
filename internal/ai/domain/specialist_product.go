package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSpecialistIDRequired           = errors.New("specialist ID is required")
	ErrProductIDRequired              = errors.New("product ID is required")
	ErrSpecialistProductNotFound      = errors.New("specialist product not found")
	ErrSpecialistProductAlreadyExists = errors.New("specialist product already exists")
)

// SpecialistProduct links a specialist to a product they can handle.
type SpecialistProduct struct {
	ID           string
	SpecialistID string
	ProductID    string
	CreatedAt    time.Time
}

// NewSpecialistProduct constructs a SpecialistProduct with validation.
func NewSpecialistProduct(id, specialistID, productID string) (*SpecialistProduct, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	if productID == "" {
		return nil, ErrProductIDRequired
	}
	return &SpecialistProduct{
		ID:           id,
		SpecialistID: specialistID,
		ProductID:    productID,
		CreatedAt:    time.Now(),
	}, nil
}

// SpecialistProductRepository defines persistence operations for SpecialistProduct.
type SpecialistProductRepository interface {
	Create(ctx context.Context, sp *SpecialistProduct) error
	Delete(ctx context.Context, specialistID, productID string) error
	FindBySpecialistID(ctx context.Context, specialistID string) ([]SpecialistProduct, error)
	FindByProductID(ctx context.Context, productID string) ([]SpecialistProduct, error)
	Exists(ctx context.Context, specialistID, productID string) (bool, error)
}
