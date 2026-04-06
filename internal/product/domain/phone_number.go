package domain

import (
	"errors"
	"time"
)

type ProductPhoneNumber struct {
	ID          string
	ProductID   string
	PhoneNumber string
	CreatedAt   time.Time
}

func NewProductPhoneNumber(id, productID, phoneNumber string) (*ProductPhoneNumber, error) {
	if productID == "" {
		return nil, errors.New("product ID is required")
	}
	if phoneNumber == "" {
		return nil, errors.New("phone number is required")
	}
	return &ProductPhoneNumber{
		ID: id, ProductID: productID, PhoneNumber: phoneNumber,
		CreatedAt: time.Now(),
	}, nil
}
