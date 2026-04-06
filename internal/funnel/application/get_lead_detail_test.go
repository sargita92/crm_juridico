package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupLeadDetailTest() (*GetLeadDetailUseCase, *mockLeadRepo, *mockLeadMovementRepo, *mockFunnelRepo, *mockColumnRepo, *mockContactProvider, *mockMessageProvider, *mockLeadNoteRepo) {
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	contactProvider := newMockContactProvider()
	messageProvider := newMockMessageProvider()
	noteRepo := newMockLeadNoteRepo()

	uc := NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo)
	return uc, leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo
}

func TestGetLeadDetail_Success_Enriched(t *testing.T) {
	uc, leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo := setupLeadDetailTest()

	// Setup funnel + column
	funnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Vendas", "")
	_ = funnelRepo.Create(context.Background(), funnel)
	col, _ := domain.NewColumn(uuid.New().String(), funnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)

	// Setup lead
	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", funnel.ID, col.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	// Setup movement
	mv := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", col.ID)
	_ = movementRepo.Create(context.Background(), mv)

	// Setup contact
	contactProvider.contacts["contact-1"] = domain.ContactInfo{Name: "Joao Silva", Phone: "+5511999990000"}

	// Setup messages
	messageProvider.messages["conv-1"] = []domain.MessageSummary{
		{Direction: "incoming", Content: "Ola, preciso de ajuda", Timestamp: time.Now()},
		{Direction: "outgoing", Content: "Como posso ajudar?", Timestamp: time.Now()},
	}

	// Setup note
	note, _ := domain.NewLeadNote(uuid.New().String(), lead.ID, "tenant-1", "Ligar amanha", "user-1")
	_ = noteRepo.Create(context.Background(), note)

	output, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: lead.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, lead.ID, output.ID)
	assert.Equal(t, "Joao Silva", output.ContactName)
	assert.Equal(t, "+5511999990000", output.ContactPhone)
	assert.Equal(t, "Vendas", output.FunnelName)
	assert.Equal(t, "Novo", output.ColumnName)
	assert.Len(t, output.Messages, 2)
	assert.Equal(t, "incoming", output.Messages[0].Direction)
	assert.Len(t, output.Movements, 1)
	assert.Equal(t, "Novo", output.Movements[0].ToColumnName)
	assert.Len(t, output.Notes, 1)
	assert.Equal(t, "Ligar amanha", output.Notes[0].Content)
}

func TestGetLeadDetail_NotFound(t *testing.T) {
	uc, _, _, _, _, _, _, _ := setupLeadDetailTest()

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: "nope",
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGetLeadDetail_WrongTenant(t *testing.T) {
	uc, leadRepo, _, funnelRepo, columnRepo, _, _, _ := setupLeadDetailTest()

	funnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Vendas", "")
	_ = funnelRepo.Create(context.Background(), funnel)
	col, _ := domain.NewColumn(uuid.New().String(), funnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), col)
	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", funnel.ID, col.ID, "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "other", LeadID: lead.ID,
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}
