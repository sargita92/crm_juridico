package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestToggleFunnel_Deactivate(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewToggleFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)
	assert.True(t, f.Active)

	output, err := uc.Execute(context.Background(), ToggleFunnelInput{TenantID: "tenant-1", FunnelID: f.ID})
	require.NoError(t, err)
	assert.False(t, output.Active)
}

func TestToggleFunnel_Activate(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewToggleFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.Deactivate()
	_ = funnelRepo.Create(context.Background(), f)

	output, err := uc.Execute(context.Background(), ToggleFunnelInput{TenantID: "tenant-1", FunnelID: f.ID})
	require.NoError(t, err)
	assert.True(t, output.Active)
}

func TestToggleFunnel_NotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewToggleFunnelUseCase(funnelRepo)

	_, err := uc.Execute(context.Background(), ToggleFunnelInput{TenantID: "tenant-1", FunnelID: "nope"})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestToggleFunnel_WrongTenant(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	uc := NewToggleFunnelUseCase(funnelRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), ToggleFunnelInput{TenantID: "other", FunnelID: f.ID})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}
