package application

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type GenerateRecurringPayments struct {
	payments domain.PaymentRepository
	billing  domain.TenantBillingRepository
	cal      domain.HolidayCalendar
	idGen    IDGenerator
	clock    Clock
}

func NewGenerateRecurringPayments(payments domain.PaymentRepository, billing domain.TenantBillingRepository, cal domain.HolidayCalendar, idGen IDGenerator, clock Clock) *GenerateRecurringPayments {
	return &GenerateRecurringPayments{payments: payments, billing: billing, cal: cal, idGen: idGen, clock: clock}
}

func (uc *GenerateRecurringPayments) Execute(ctx context.Context) (int, error) {
	ctx, span := tracer.Start(ctx, "GenerateRecurringPayments")
	defer span.End()
	today := truncateDay(uc.clock.Now())
	span.SetAttributes(attribute.String("today", today.Format("2006-01-02")))
	tenants, err := uc.billing.ListActiveBillable(ctx)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, t := range tenants {
		if t.Config.DataInicioCobranca == nil || t.Config.ValorCents == nil || t.Config.DiaVencimento == nil {
			continue
		}
		start := *t.Config.DataInicioCobranca
		if start.After(today) {
			continue
		}
		comps := computeCompetencias(start, today, t.Config.Plano)
		for _, comp := range comps {
			exists, err := uc.payments.ExistsRecorrente(ctx, t.TenantID, comp)
			if err != nil {
				return created, err
			}
			if exists {
				continue
			}
			day := int(*t.Config.DiaVencimento)
			venc := time.Date(comp.Year(), comp.Month(), day, 0, 0, 0, 0, time.UTC)
			venc = uc.cal.NextBusinessDay(venc)
			p, err := domain.NewRecorrentePayment(uc.idGen.NewID(), t.TenantID, *t.Config.ValorCents, comp, venc)
			if err != nil {
				return created, err
			}
			if err := uc.payments.Create(ctx, p); err != nil {
				return created, err
			}
			created++
		}
	}
	span.SetAttributes(attribute.Int("created", created))
	return created, nil
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// computeCompetencias retorna as competências (primeiro dia do mês/ano) que
// devem ter lançamento entre `start` e `today` inclusivos. Para PlanMensal
// avança 1 mês; para PlanAnual avança 1 ano.
func computeCompetencias(start, today time.Time, plano domain.Plan) []time.Time {
	out := []time.Time{}
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		out = append(out, cursor)
		if plano == domain.PlanAnual {
			cursor = cursor.AddDate(1, 0, 0)
		} else {
			cursor = cursor.AddDate(0, 1, 0)
		}
	}
	return out
}
