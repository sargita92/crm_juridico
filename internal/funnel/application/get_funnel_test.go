package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestGetFunnel_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewGetFunnelUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "desc")
	_ = funnelRepo.Create(context.Background(), f)
	col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)

	output, err := uc.Execute(context.Background(), GetFunnelInput{TenantID: "tenant-1", FunnelID: f.ID})
	require.NoError(t, err)
	assert.Equal(t, f.ID, output.ID)
	assert.Equal(t, "Pipeline", output.Name)
	assert.Len(t, output.Columns, 1)
	assert.Equal(t, "Novo", output.Columns[0].Name)
}

func TestGetFunnel_NotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewGetFunnelUseCase(funnelRepo, columnRepo)

	_, err := uc.Execute(context.Background(), GetFunnelInput{TenantID: "tenant-1", FunnelID: "nonexistent"})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestGetFunnel_WrongTenant(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewGetFunnelUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), GetFunnelInput{TenantID: "other-tenant", FunnelID: f.ID})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}
