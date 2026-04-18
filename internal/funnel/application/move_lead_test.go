package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupMoveLeadTest(t *testing.T) (*MoveLeadUseCase, *mockFunnelRepo, *mockColumnRepo, *mockLeadRepo, *mockLeadMovementRepo, *domain.Lead, *domain.Column, *domain.Column) {
	t.Helper()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	col1, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col1)
	col2, _ := domain.NewColumn(uuid.New().String(), f.ID, "Qualificado", 1, domain.ColumnTypeIntermediate, "#eab308")
	_ = columnRepo.Create(context.Background(), col2)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, col1.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	return uc, funnelRepo, columnRepo, leadRepo, movementRepo, lead, col1, col2
}

func TestMoveLead_Success(t *testing.T) {
	uc, _, _, leadRepo, movementRepo, lead, col1, col2 := setupMoveLeadTest(t)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1",
		LeadID:   lead.ID,
		ColumnID: col2.ID,
	})

	require.NoError(t, err)
	updated := leadRepo.leads[lead.ID]
	assert.Equal(t, col2.ID, updated.ColumnID)
	assert.Equal(t, domain.LeadStatusOpen, updated.Status)

	mvs := movementRepo.movements[lead.ID]
	require.Len(t, mvs, 1)
	assert.Equal(t, col1.ID, mvs[0].FromColumnID)
	assert.Equal(t, col2.ID, mvs[0].ToColumnID)
}

func TestMoveLead_ToWonColumn(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)
	entryCol, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "")
	_ = columnRepo.Create(context.Background(), entryCol)
	wonCol, _ := domain.NewColumn(uuid.New().String(), f.ID, "Ganho", 1, domain.ColumnTypeWon, "")
	_ = columnRepo.Create(context.Background(), wonCol)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", f.ID, entryCol.ID, "c-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1", LeadID: lead.ID, ColumnID: wonCol.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.LeadStatusWon, leadRepo.leads[lead.ID].Status)
}

func TestMoveLead_NotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1", LeadID: "nope", ColumnID: "col",
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestMoveLead_WrongTenant(t *testing.T) {
	uc, _, _, _, _, lead, _, col2 := setupMoveLeadTest(t)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "other", LeadID: lead.ID, ColumnID: col2.ID,
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestMoveLead_ColumnNotFound(t *testing.T) {
	uc, _, _, _, _, lead, _, _ := setupMoveLeadTest(t)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1", LeadID: lead.ID, ColumnID: "nope",
	})
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

// CT-66: Log de movimentacao de lead (tenant_id, lead_id, from_column, to_column)
func TestMoveLead_AuditLogOnSuccess(t *testing.T) {
	uc, _, _, _, _, lead, col1, col2 := setupMoveLeadTest(t)

	core, logs := observer.New(zapcore.InfoLevel)
	uc.SetAuditLogger(zap.New(core))

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1", LeadID: lead.ID, ColumnID: col2.ID,
	})
	require.NoError(t, err)

	entries := logs.FilterMessage("lead moved").All()
	require.Len(t, entries, 1, "deve registrar info audit na movimentacao")
	fields := entries[0].ContextMap()
	assert.Equal(t, "tenant-1", fields["tenant_id"])
	assert.Equal(t, lead.ID, fields["lead_id"])
	assert.Equal(t, col1.ID, fields["from_column_id"])
	assert.Equal(t, col2.ID, fields["to_column_id"])
}

// CT-65: Log de acesso negado (tenant B tenta mover lead de tenant A)
func TestMoveLead_CrossTenantAccessIsLogged(t *testing.T) {
	uc, _, _, _, _, lead, _, col2 := setupMoveLeadTest(t)

	core, logs := observer.New(zapcore.WarnLevel)
	uc.SetAuditLogger(zap.New(core))

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-b", LeadID: lead.ID, ColumnID: col2.ID,
	})
	require.ErrorIs(t, err, domain.ErrLeadNotFound)

	entries := logs.FilterMessage("cross-tenant lead access denied").All()
	require.Len(t, entries, 1, "deve registrar warning ao negar movimentacao cross-tenant")
	fields := entries[0].ContextMap()
	assert.Equal(t, "tenant-b", fields["tenant_id"])
	assert.Equal(t, lead.ID, fields["lead_id"])
	assert.Equal(t, "move_lead", fields["operation"])
}

func TestMoveLead_DifferentFunnel(t *testing.T) {
	uc, funnelRepo, columnRepo, leadRepo, _, lead, _, _ := setupMoveLeadTest(t)

	f2, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline 2", "")
	_ = funnelRepo.Create(context.Background(), f2)
	col, _ := domain.NewColumn(uuid.New().String(), f2.ID, "Novo", 0, domain.ColumnTypeEntry, "")
	_ = columnRepo.Create(context.Background(), col)

	err := uc.Execute(context.Background(), MoveLeadInput{
		TenantID: "tenant-1", LeadID: lead.ID,
		ColumnID: col.ID, FunnelID: f2.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, f2.ID, leadRepo.leads[lead.ID].FunnelID)
}
