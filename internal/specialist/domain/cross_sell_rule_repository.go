package domain

import "context"

type CrossSellRuleRepository interface {
	Save(ctx context.Context, rule *CrossSellRule) error
	FindByID(ctx context.Context, id string) (*CrossSellRule, error)
	ListBySpecialistID(ctx context.Context, specialistID string) ([]*CrossSellRule, error)
	ListActiveBySpecialistOrdered(ctx context.Context, specialistID string) ([]*CrossSellRule, error)
	Delete(ctx context.Context, id string) error
}
