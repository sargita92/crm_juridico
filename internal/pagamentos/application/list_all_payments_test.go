package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

func TestListAllPayments_WithoutTenantFilter_ReturnsAll(t *testing.T) {
	repo := newFakeRepo()
	repo.listAllResult = &domain.ListResult{
		Items: []domain.Payment{},
		Total: 10,
		Page:  1, PageSize: 20,
	}
	uc := application.NewListAllPayments(repo)

	out, err := uc.Execute(context.Background(), application.ListAllInput{Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(10), out.Total)
	assert.Equal(t, "", repo.lastListAllFilters.TenantID)
}

func TestListAllPayments_WithTenantFilter_ForwardsTenant(t *testing.T) {
	repo := newFakeRepo()
	repo.listAllResult = &domain.ListResult{Items: nil, Total: 0, Page: 1, PageSize: 20}
	uc := application.NewListAllPayments(repo)
	_, err := uc.Execute(context.Background(), application.ListAllInput{TenantID: "t9", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, "t9", repo.lastListAllFilters.TenantID)
}
