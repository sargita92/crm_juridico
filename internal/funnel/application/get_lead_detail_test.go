package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestGetLeadDetail_Success(t *testing.T) {
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewGetLeadDetailUseCase(leadRepo, movementRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)
	mv := domain.NewLeadMovement(uuid.New().String(), lead.ID, "", "col-1")
	_ = movementRepo.Create(context.Background(), mv)

	output, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: lead.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, lead.ID, output.ID)
	assert.Equal(t, "open", output.Status)
	assert.Len(t, output.Movements, 1)
	assert.Equal(t, "", output.Movements[0].FromColumnID)
	assert.Equal(t, "col-1", output.Movements[0].ToColumnID)
}

func TestGetLeadDetail_NotFound(t *testing.T) {
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewGetLeadDetailUseCase(leadRepo, movementRepo)

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "tenant-1", LeadID: "nope",
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGetLeadDetail_WrongTenant(t *testing.T) {
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewGetLeadDetailUseCase(leadRepo, movementRepo)

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = leadRepo.Create(context.Background(), lead)

	_, err := uc.Execute(context.Background(), GetLeadDetailInput{
		TenantID: "other", LeadID: lead.ID,
	})
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}
