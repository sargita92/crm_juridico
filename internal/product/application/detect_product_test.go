package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

func TestDetectProduct_MatchesKeyword(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewDetectProductUseCase(repo)

	product, err := domain.NewProduct("prod-1", "tenant-1", "Consultoria Trabalhista", "", []string{"trabalhista", "CLT"})
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), product))

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de ajuda com processo trabalhista")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "prod-1", productID)
}

func TestDetectProduct_NoMatch(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewDetectProductUseCase(repo)

	product, err := domain.NewProduct("prod-1", "tenant-1", "Consultoria Trabalhista", "", []string{"trabalhista", "CLT"})
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), product))

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de um advogado de familia")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, productID)
}

func TestDetectProduct_CaseInsensitive(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewDetectProductUseCase(repo)

	product, err := domain.NewProduct("prod-1", "tenant-1", "Consultoria", "", []string{"CLT"})
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), product))

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "tenho duvidas sobre clt")

	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "prod-1", productID)
}

func TestDetectProduct_InactiveProductIgnored(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewDetectProductUseCase(repo)

	product, err := domain.NewProduct("prod-1", "tenant-1", "Consultoria Trabalhista", "", []string{"trabalhista"})
	require.NoError(t, err)
	product.Deactivate()
	require.NoError(t, repo.Create(context.Background(), product))

	productID, found, err := uc.Execute(context.Background(), "tenant-1", "preciso de ajuda trabalhista")

	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, productID)
}
