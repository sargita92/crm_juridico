package domain

import "context"

type StepRepository interface {
	Create(ctx context.Context, step *Step) error
	FindByID(ctx context.Context, id string) (*Step, error)
	Update(ctx context.Context, step *Step) error
	Delete(ctx context.Context, id string) error
	FindBySpecialistID(ctx context.Context, specialistID string) ([]Step, error)
	GetMaxOrderIndex(ctx context.Context, specialistID string) (int, error)
	ReorderAfterDelete(ctx context.Context, specialistID string, deletedOrder int) error
	SwapOrder(ctx context.Context, step1ID string, order1 int, step2ID string, order2 int) error
}
