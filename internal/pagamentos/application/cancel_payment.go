package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type CancelPayment struct {
	repo  domain.PaymentRepository
	clock Clock
}

func NewCancelPayment(repo domain.PaymentRepository, clock Clock) *CancelPayment {
	return &CancelPayment{repo: repo, clock: clock}
}

func (uc *CancelPayment) Execute(ctx context.Context, paymentID, userID, motivo string) error {
	p, err := uc.repo.FindByIDAdmin(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := p.Cancel(userID, motivo); err != nil {
		return err
	}
	return uc.repo.Update(ctx, p)
}
