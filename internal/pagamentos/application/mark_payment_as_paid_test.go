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

func TestMarkPaymentAsPaid_Success(t *testing.T) {
	repo := newFakeRepo()
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	repo.payments["p1"] = p

	clk := &fakeClock{now: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)}
	uc := application.NewMarkPaymentAsPaid(repo, clk)

	err := uc.Execute(context.Background(), "p1", "user-1")
	require.NoError(t, err)

	saved := repo.payments["p1"]
	assert.Equal(t, domain.StatusPago, saved.Status)
	require.NotNil(t, saved.PaidByUserID)
	assert.Equal(t, "user-1", *saved.PaidByUserID)
	require.NotNil(t, saved.DataPagamento)
	assert.Equal(t, clk.now, *saved.DataPagamento)
}

func TestMarkPaymentAsPaid_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewMarkPaymentAsPaid(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "missing", "user-1")
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestMarkPaymentAsPaid_AlreadyPaid_ReturnsInvalidTransition(t *testing.T) {
	repo := newFakeRepo()
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	_ = p.MarkAsPaid("u0", time.Now())
	repo.payments["p1"] = p

	uc := application.NewMarkPaymentAsPaid(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "p1", "user-1")
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}
