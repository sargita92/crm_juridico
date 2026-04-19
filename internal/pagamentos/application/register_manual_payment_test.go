package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestRegisterManualPayment_Success(t *testing.T) {
	repo := newFakeRepo()
	clk := &fakeClock{now: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}
	uc := application.NewRegisterManualPayment(repo, &fakeIDGen{id: "gen-1"}, clk)

	out, err := uc.Execute(context.Background(), application.RegisterManualPaymentInput{
		TenantID:       "t1",
		Descricao:      "Setup",
		ValorCents:     15000,
		DataVencimento: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		Observacao:     "obs",
	})
	require.NoError(t, err)
	assert.Equal(t, "gen-1", out.ID)

	saved := repo.payments["gen-1"]
	require.NotNil(t, saved)
	assert.Equal(t, domain.StatusPendente, saved.Status)
	assert.Equal(t, domain.TypeAvulso, saved.Tipo)
	assert.Equal(t, int64(15000), saved.ValorCents)
	assert.Equal(t, "Setup", saved.Descricao)
	assert.Equal(t, "obs", saved.Observacao)
	assert.Equal(t, clk.now, saved.CreatedAt)
	assert.Equal(t, clk.now, saved.UpdatedAt)
}

func TestRegisterManualPayment_RejectsZeroValor(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewRegisterManualPayment(repo, &fakeIDGen{id: "x"}, &fakeClock{})
	_, err := uc.Execute(context.Background(), application.RegisterManualPaymentInput{
		TenantID:       "t1",
		Descricao:      "X",
		ValorCents:     0,
		DataVencimento: time.Now(),
	})
	assert.ErrorIs(t, err, domain.ErrValorInvalido)
	assert.Empty(t, repo.payments)
}

func TestRegisterManualPayment_RejectsEmptyTenant(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewRegisterManualPayment(repo, &fakeIDGen{id: "x"}, &fakeClock{})
	_, err := uc.Execute(context.Background(), application.RegisterManualPaymentInput{
		TenantID:       "",
		Descricao:      "X",
		ValorCents:     1000,
		DataVencimento: time.Now(),
	})
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}

func TestRegisterManualPayment_RejectsEmptyDescricao(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewRegisterManualPayment(repo, &fakeIDGen{id: "x"}, &fakeClock{})
	_, err := uc.Execute(context.Background(), application.RegisterManualPaymentInput{
		TenantID:       "t1",
		Descricao:      "",
		ValorCents:     1000,
		DataVencimento: time.Now(),
	})
	assert.ErrorIs(t, err, domain.ErrDescricaoRequired)
}

func TestRegisterManualPayment_PropagatesRepoError(t *testing.T) {
	repo := newFakeRepo()
	boom := errors.New("boom")
	repo.createErr = boom
	uc := application.NewRegisterManualPayment(repo, &fakeIDGen{id: "x"}, &fakeClock{now: time.Now()})
	_, err := uc.Execute(context.Background(), application.RegisterManualPaymentInput{
		TenantID:       "t1",
		Descricao:      "X",
		ValorCents:     1000,
		DataVencimento: time.Now(),
	})
	assert.ErrorIs(t, err, boom)
}
