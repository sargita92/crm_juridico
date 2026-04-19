package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestRefreshOverdue_MarksPastDueAsAtrasado(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	venc := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC) // sexta
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, venc, "")
	repo.payments["p1"] = p
	repo.overdueResult = []domain.Payment{*p}

	uc := application.NewRefreshOverdueStatuses(repo, cal, 1, &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)})
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, domain.StatusAtrasado, repo.payments["p1"].Status)
}

func TestRefreshOverdue_RespectsGracePeriod(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	venc := time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC) // segunda
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, venc, "")
	repo.payments["p1"] = p
	repo.overdueResult = []domain.Payment{*p}

	uc := application.NewRefreshOverdueStatuses(repo, cal, 2, &fakeClock{now: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)})
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, domain.StatusPendente, repo.payments["p1"].Status)
}

func TestRefreshOverdue_EmptyList_ReturnsZero(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	uc := application.NewRefreshOverdueStatuses(repo, cal, 1, &fakeClock{now: time.Now()})
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestRefreshOverdue_IgnoresNonPendente(t *testing.T) {
	cal := domain.NewBrazilHolidayCalendar()
	repo := newFakeRepo()
	venc := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, venc, "")
	_ = p.MarkAsPaid("u0", time.Now()) // status pago
	repo.payments["p1"] = p
	repo.overdueResult = []domain.Payment{*p}
	uc := application.NewRefreshOverdueStatuses(repo, cal, 1, &fakeClock{now: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)})
	n, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, domain.StatusPago, repo.payments["p1"].Status)
}
