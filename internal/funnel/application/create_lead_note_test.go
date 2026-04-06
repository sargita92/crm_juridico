package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestCreateLeadNote_Success(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	output, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    lead.ID,
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	require.NoError(t, err)
	assert.Equal(t, "Ligar amanha", output.Content)
	assert.Equal(t, "user-1", output.CreatedBy)
	assert.NotEmpty(t, output.ID)
}

func TestCreateLeadNote_LeadNotFound(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    "nope",
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestCreateLeadNote_WrongTenant(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "other-tenant",
		LeadID:    lead.ID,
		Content:   "Ligar amanha",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestCreateLeadNote_EmptyContent(t *testing.T) {
	leadRepo := newMockLeadRepo()
	noteRepo := newMockLeadNoteRepo()
	uc := NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), CreateLeadNoteInput{
		TenantID:  "tenant-1",
		LeadID:    lead.ID,
		Content:   "",
		CreatedBy: "user-1",
	})

	assert.ErrorIs(t, err, domain.ErrNoteContentRequired)
}
