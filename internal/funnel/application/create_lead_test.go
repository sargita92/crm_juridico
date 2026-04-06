package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupCreateLeadTest(t *testing.T) (*CreateLeadUseCase, *mockFunnelRepo, *mockColumnRepo, *mockLeadRepo, *mockLeadMovementRepo, *domain.Funnel, *domain.Column) {
	t.Helper()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.SetDefault()
	_ = funnelRepo.Create(context.Background(), f)

	entryCol, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), entryCol)

	return uc, funnelRepo, columnRepo, leadRepo, movementRepo, f, entryCol
}

func TestCreateLead_Success(t *testing.T) {
	uc, _, _, leadRepo, movementRepo, f, entryCol := setupCreateLeadTest(t)

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
	})

	require.NoError(t, err)
	assert.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, f.ID, l.FunnelID)
		assert.Equal(t, entryCol.ID, l.ColumnID)
		assert.Equal(t, "contact-1", l.ContactID)
		assert.Equal(t, domain.LeadStatusOpen, l.Status)
	}
	assert.Len(t, movementRepo.movements, 1)
}

func TestCreateLead_AlreadyExists_Noop(t *testing.T) {
	uc, _, _, leadRepo, _, _, _ := setupCreateLeadTest(t)

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID: "tenant-1", ContactID: "contact-1", ConversationID: "conv-1",
	})
	require.NoError(t, err)

	err = uc.Execute(context.Background(), CreateLeadInput{
		TenantID: "tenant-1", ContactID: "contact-1", ConversationID: "conv-2",
	})
	require.NoError(t, err)
	assert.Len(t, leadRepo.leads, 1, "should not create duplicate lead")
}

func TestCreateLead_NoDefaultFunnel(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID: "tenant-1", ContactID: "contact-1", ConversationID: "conv-1",
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestCreateLead_CreateFromConversation(t *testing.T) {
	uc, _, _, leadRepo, _, _, _ := setupCreateLeadTest(t)

	err := uc.CreateFromConversation(context.Background(), "tenant-1", "contact-1", "conv-1")
	require.NoError(t, err)
	assert.Len(t, leadRepo.leads, 1)
}
