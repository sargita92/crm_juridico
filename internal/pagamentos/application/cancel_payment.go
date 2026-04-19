package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

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
	ctx, span := tracer.Start(ctx, "CancelPayment")
	defer span.End()
	span.SetAttributes(attribute.String("payment_id", paymentID), attribute.String("user_id", userID))
	p, err := uc.repo.FindByIDAdmin(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := p.Cancel(userID, motivo); err != nil {
		return err
	}
	return uc.repo.Update(ctx, p)
}
