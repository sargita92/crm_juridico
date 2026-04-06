package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

func TestCreateProduct_Success(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewCreateProductUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateProductInput{
		TenantID:    "tenant-1",
		Name:        "Consultoria Trabalhista",
		Description: "Servico de consultoria",
		Keywords:    []string{"trabalhista", "CLT"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Consultoria Trabalhista", output.Name)
	assert.Equal(t, "Servico de consultoria", output.Description)
	assert.Equal(t, []string{"trabalhista", "CLT"}, output.Keywords)
	assert.True(t, output.Active)
}

func TestCreateProduct_EmptyName_Error(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewCreateProductUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateProductInput{
		TenantID: "tenant-1",
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrProductNameRequired)
}

func TestCreateProduct_EmptyTenantID_Error(t *testing.T) {
	repo := newMockProductRepo()
	uc := NewCreateProductUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateProductInput{
		TenantID: "",
		Name:     "Produto",
	})

	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}
