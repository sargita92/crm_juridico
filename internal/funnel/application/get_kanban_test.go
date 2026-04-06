package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupKanbanTest(t *testing.T) (*GetKanbanUseCase, *mockFunnelRepo, *mockColumnRepo, *mockLeadRepo, *domain.Funnel) {
	t.Helper()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.SetDefault()
	_ = funnelRepo.Create(context.Background(), f)

	for i, dc := range domain.DefaultColumns {
		col, _ := domain.NewColumn(uuid.New().String(), f.ID, dc.Name, i, dc.Type, dc.Color)
		_ = columnRepo.Create(context.Background(), col)
	}

	return uc, funnelRepo, columnRepo, leadRepo, f
}

func TestGetKanban_DefaultFunnel(t *testing.T) {
	uc, _, _, _, f := setupKanbanTest(t)

	output, err := uc.Execute(context.Background(), GetKanbanInput{
		TenantID: "tenant-1",
	})

	require.NoError(t, err)
	assert.Equal(t, f.ID, output.FunnelID)
	assert.Equal(t, "Pipeline", output.FunnelName)
	assert.Len(t, output.Columns, len(domain.DefaultColumns))
	assert.Equal(t, 0, output.TotalLeads)
}

func TestGetKanban_SpecificFunnel(t *testing.T) {
	uc, _, _, _, f := setupKanbanTest(t)

	output, err := uc.Execute(context.Background(), GetKanbanInput{
		TenantID: "tenant-1",
		FunnelID: f.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, f.ID, output.FunnelID)
}

func TestGetKanban_WithLeads(t *testing.T) {
	uc, _, columnRepo, leadRepo, f := setupKanbanTest(t)

	// Get the entry column
	var entryColID string
	for _, col := range columnRepo.byFunnel[f.ID] {
		if col.Type == domain.ColumnTypeEntry {
			entryColID = col.ID
			break
		}
	}

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, entryColID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	output, err := uc.Execute(context.Background(), GetKanbanInput{TenantID: "tenant-1"})
	require.NoError(t, err)
	assert.Equal(t, 1, output.TotalLeads)
}

func TestGetKanban_FunnelNotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)

	_, err := uc.Execute(context.Background(), GetKanbanInput{TenantID: "tenant-1"})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestGetKanban_WrongTenant(t *testing.T) {
	uc, _, _, _, f := setupKanbanTest(t)

	_, err := uc.Execute(context.Background(), GetKanbanInput{
		TenantID: "other", FunnelID: f.ID,
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}
