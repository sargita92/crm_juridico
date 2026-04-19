package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type MarkPaymentAsPaid struct {
	repo  domain.PaymentRepository
	clock Clock
}

func NewMarkPaymentAsPaid(repo domain.PaymentRepository, clock Clock) *MarkPaymentAsPaid {
	return &MarkPaymentAsPaid{repo: repo, clock: clock}
}

func (uc *MarkPaymentAsPaid) Execute(ctx context.Context, paymentID, userID string) error {
	ctx, span := tracer.Start(ctx, "MarkPaymentAsPaid")
	defer span.End()
	span.SetAttributes(attribute.String("payment_id", paymentID), attribute.String("user_id", userID))
	p, err := uc.repo.FindByIDAdmin(ctx, paymentID)
	if err != nil {
		return err
	}
	if err := p.MarkAsPaid(userID, uc.clock.Now()); err != nil {
		return err
	}
	return uc.repo.Update(ctx, p)
}
