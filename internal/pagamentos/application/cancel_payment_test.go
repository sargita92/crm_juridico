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

func TestCancelPayment_Success(t *testing.T) {
	repo := newFakeRepo()
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	repo.payments["p1"] = p

	uc := application.NewCancelPayment(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "p1", "user-1", "Duplicado")
	require.NoError(t, err)

	saved := repo.payments["p1"]
	assert.Equal(t, domain.StatusCancelado, saved.Status)
	require.NotNil(t, saved.CancelledByUserID)
	assert.Equal(t, "user-1", *saved.CancelledByUserID)
	require.NotNil(t, saved.CancelledAt)
	assert.Equal(t, "Duplicado", saved.Observacao)
}

func TestCancelPayment_RequiresMotivo(t *testing.T) {
	repo := newFakeRepo()
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	repo.payments["p1"] = p

	uc := application.NewCancelPayment(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "p1", "user-1", "")
	assert.ErrorIs(t, err, domain.ErrMotivoRequired)
}

func TestCancelPayment_NotFound(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewCancelPayment(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "missing", "user-1", "Motivo")
	assert.ErrorIs(t, err, domain.ErrPaymentNotFound)
}

func TestCancelPayment_AlreadyPaid_ReturnsInvalidTransition(t *testing.T) {
	repo := newFakeRepo()
	p, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	_ = p.MarkAsPaid("u0", time.Now())
	repo.payments["p1"] = p

	uc := application.NewCancelPayment(repo, &fakeClock{})
	err := uc.Execute(context.Background(), "p1", "user-1", "Motivo")
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)
}
