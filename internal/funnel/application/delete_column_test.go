package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestDeleteColumn_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)
	col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Col", 0, domain.ColumnTypeIntermediate, "")
	_ = columnRepo.Create(context.Background(), col)

	err := uc.Execute(context.Background(), DeleteColumnInput{TenantID: "tenant-1", ColumnID: col.ID})
	require.NoError(t, err)
	assert.Len(t, columnRepo.columns, 0)
}

func TestDeleteColumn_HasLeads_Error(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)
	col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Col", 0, domain.ColumnTypeEntry, "")
	_ = columnRepo.Create(context.Background(), col)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, col.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	err := uc.Execute(context.Background(), DeleteColumnInput{TenantID: "tenant-1", ColumnID: col.ID})
	assert.ErrorIs(t, err, domain.ErrColumnHasLeads)
}

func TestDeleteColumn_NotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)

	err := uc.Execute(context.Background(), DeleteColumnInput{TenantID: "tenant-1", ColumnID: "nope"})
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestDeleteColumn_WrongTenant(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	uc := NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)
	col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Col", 0, domain.ColumnTypeIntermediate, "")
	_ = columnRepo.Create(context.Background(), col)

	err := uc.Execute(context.Background(), DeleteColumnInput{TenantID: "other", ColumnID: col.ID})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}
