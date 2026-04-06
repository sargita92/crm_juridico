package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestListFunnels_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Funil 1", "desc")
	_ = funnelRepo.Create(context.Background(), f)
	col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)

	items, err := uc.Execute(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Funil 1", items[0].Name)
	assert.Equal(t, 1, items[0].ColumnCount)
	assert.Equal(t, 0, items[0].LeadCount)
}

func TestListFunnels_Empty(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)

	items, err := uc.Execute(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Len(t, items, 0)
}
