package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestCreateFunnel_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateFunnelUseCase(funnelRepo, columnRepo)

	output, err := uc.Execute(context.Background(), CreateFunnelInput{
		TenantID:    "tenant-1",
		Name:        "Pipeline Principal",
		Description: "Funil padrao",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Pipeline Principal", output.Name)
	assert.Equal(t, "Funil padrao", output.Description)
	assert.True(t, output.Active)
	assert.True(t, output.IsDefault, "first funnel should be default")
	assert.Len(t, columnRepo.columns, len(domain.DefaultColumns))
}

func TestCreateFunnel_SecondFunnel_NotDefault(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateFunnelUseCase(funnelRepo, columnRepo)

	_, err := uc.Execute(context.Background(), CreateFunnelInput{
		TenantID: "tenant-1",
		Name:     "Primeiro",
	})
	require.NoError(t, err)

	output, err := uc.Execute(context.Background(), CreateFunnelInput{
		TenantID: "tenant-1",
		Name:     "Segundo",
	})
	require.NoError(t, err)
	assert.False(t, output.IsDefault, "second funnel should not be default")
}

func TestCreateFunnel_EmptyName_Error(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateFunnelUseCase(funnelRepo, columnRepo)

	_, err := uc.Execute(context.Background(), CreateFunnelInput{
		TenantID: "tenant-1",
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelNameRequired)
}

func TestCreateFunnel_EmptyTenantID_Error(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateFunnelUseCase(funnelRepo, columnRepo)

	_, err := uc.Execute(context.Background(), CreateFunnelInput{
		TenantID: "",
		Name:     "Funil",
	})

	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}
