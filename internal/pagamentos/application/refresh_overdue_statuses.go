package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

type RefreshOverdueStatuses struct {
	repo  domain.PaymentRepository
	cal   domain.HolidayCalendar
	grace int
	clock Clock
}

func NewRefreshOverdueStatuses(repo domain.PaymentRepository, cal domain.HolidayCalendar, graceDays int, clock Clock) *RefreshOverdueStatuses {
	return &RefreshOverdueStatuses{repo: repo, cal: cal, grace: graceDays, clock: clock}
}

func (uc *RefreshOverdueStatuses) Execute(ctx context.Context) (int, error) {
	today := truncateDay(uc.clock.Now())
	candidates, err := uc.repo.ListOverdueCandidates(ctx, today)
	if err != nil {
		return 0, err
	}
	updated := 0
	for i := range candidates {
		p := candidates[i]
		if p.Status != domain.StatusPendente {
			continue
		}
		deadline := uc.cal.AddBusinessDays(p.DataVencimento, uc.grace)
		if today.After(deadline) {
			p.Status = domain.StatusAtrasado
			p.UpdatedAt = uc.clock.Now()
			if err := uc.repo.Update(ctx, &p); err != nil {
				return updated, err
			}
			updated++
		}
	}
	return updated, nil
}
