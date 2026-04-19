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

func TestListTenantPayments_ForwardsFiltersAndReturnsResult(t *testing.T) {
	repo := newFakeRepo()
	p1, _ := domain.NewAvulsoPayment("p1", "t1", "X", 1000, time.Now(), "")
	p2, _ := domain.NewAvulsoPayment("p2", "t1", "Y", 2000, time.Now(), "")
	repo.listResult = &domain.ListResult{
		Items:    []domain.Payment{*p1, *p2},
		Total:    2,
		Page:     1,
		PageSize: 20,
	}

	uc := application.NewListTenantPayments(repo)
	status := domain.StatusPendente
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	out, err := uc.Execute(context.Background(), application.ListTenantInput{
		TenantID: "t1", Status: &status, DataInicial: &from, DataFinal: &to,
		Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), out.Total)
	assert.Len(t, out.Items, 2)
	assert.Equal(t, "t1", repo.lastListFilters.TenantID)
	assert.Equal(t, &status, repo.lastListFilters.Status)
	assert.Equal(t, &from, repo.lastListFilters.DataInicial)
	assert.Equal(t, &to, repo.lastListFilters.DataFinal)
	assert.Equal(t, 1, repo.lastListFilters.Page)
	assert.Equal(t, 20, repo.lastListFilters.PageSize)
}

func TestListTenantPayments_PropagatesError(t *testing.T) {
	repo := newFakeRepo()
	repo.listErr = errBoom
	uc := application.NewListTenantPayments(repo)
	_, err := uc.Execute(context.Background(), application.ListTenantInput{TenantID: "t1"})
	assert.ErrorIs(t, err, errBoom)
}
