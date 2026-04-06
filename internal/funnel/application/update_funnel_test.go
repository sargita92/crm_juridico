package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestUpdateFunnel_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewUpdateFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Old Name", "old desc")
	_ = funnelRepo.Create(context.Background(), f)

	output, err := uc.Execute(context.Background(), UpdateFunnelInput{
		TenantID:    "tenant-1",
		FunnelID:    f.ID,
		Name:        "New Name",
		Description: "new desc",
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", output.Name)
	assert.Equal(t, "new desc", output.Description)
}

func TestUpdateFunnel_NotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewUpdateFunnelUseCase(funnelRepo)

	_, err := uc.Execute(context.Background(), UpdateFunnelInput{
		TenantID: "tenant-1",
		FunnelID: "nonexistent",
		Name:     "Name",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestUpdateFunnel_WrongTenant(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewUpdateFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Name", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), UpdateFunnelInput{
		TenantID: "other-tenant",
		FunnelID: f.ID,
		Name:     "Updated",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestUpdateFunnel_EmptyName(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewUpdateFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Name", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), UpdateFunnelInput{
		TenantID: "tenant-1",
		FunnelID: f.ID,
		Name:     "",
	})

	assert.ErrorIs(t, err, domain.ErrFunnelNameRequired)
}
