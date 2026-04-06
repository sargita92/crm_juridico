package domain

import (
	"errors"
	"time"
)

type FunnelProduct struct {
	ID        string
	FunnelID  string
	ProductID string
	Priority  int
	CreatedAt time.Time
}

func NewFunnelProduct(id, funnelID, productID string, priority int) (*FunnelProduct, error) {
	if funnelID == "" {
		return nil, errors.New("funnel ID is required")
	}
	if productID == "" {
		return nil, errors.New("product ID is required")
	}
	if priority < 1 {
		priority = 1
	}
	return &FunnelProduct{
		ID: id, FunnelID: funnelID, ProductID: productID,
		Priority: priority, CreatedAt: time.Now(),
	}, nil
}
