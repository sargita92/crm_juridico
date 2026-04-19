package application

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type RegisterManualPaymentInput struct {
	TenantID       string
	Descricao      string
	ValorCents     int64
	DataVencimento time.Time
	Observacao     string
}

type RegisterManualPaymentOutput struct {
	ID string
}

type RegisterManualPayment struct {
	repo  domain.PaymentRepository
	idGen IDGenerator
	clock Clock
}

func NewRegisterManualPayment(repo domain.PaymentRepository, idGen IDGenerator, clock Clock) *RegisterManualPayment {
	return &RegisterManualPayment{repo: repo, idGen: idGen, clock: clock}
}

func (uc *RegisterManualPayment) Execute(ctx context.Context, in RegisterManualPaymentInput) (*RegisterManualPaymentOutput, error) {
	ctx, span := tracer.Start(ctx, "RegisterManualPayment")
	defer span.End()
	span.SetAttributes(
		attribute.String("tenant_id", in.TenantID),
		attribute.Int64("valor_cents", in.ValorCents),
	)
	p, err := domain.NewAvulsoPayment(uc.idGen.NewID(), in.TenantID, in.Descricao, in.ValorCents, in.DataVencimento, in.Observacao)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = uc.clock.Now()
	p.UpdatedAt = p.CreatedAt
	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return &RegisterManualPaymentOutput{ID: p.ID}, nil
}
