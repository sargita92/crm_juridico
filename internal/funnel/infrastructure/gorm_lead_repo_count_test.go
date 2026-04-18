package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// seedLeadForUser persists a lead directly via the leadModel, wiring the
// supplied responsible_user_id and status. It reuses the existing fixture
// helpers (newLeadFixture, createContact, createConversation) to keep FK
// constraints satisfied without requiring a real users row — leads.responsible_user_id
// is a nullable char(36) column without a FK (see migration 000043).
func seedLeadForUser(t *testing.T, db *gorm.DB, fx *leadFixture, userID string, status domain.LeadStatus) {
	t.Helper()
	contactID := createContact(t, db, fx.tenantID, "Cliente "+uuid.New().String()[:8], "+55119"+uuid.New().String()[:8])
	convID := createConversation(t, db, fx.tenantID, contactID)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, contactID, convID)
	require.NoError(t, err)
	lead.AssignResponsible(userID)
	lead.Status = status

	require.NoError(t, db.Create(leadToModel(lead)).Error)
}

func TestGormLeadRepository_CountActiveByUsers(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()

	fxA := newLeadFixture(t, repos, db)
	fxB := newLeadFixture(t, repos, db) // different tenant

	userA := uuid.New().String()
	userB := uuid.New().String()
	userC := uuid.New().String() // will have zero open leads in tenant A

	// Tenant A: 3 open for userA, 1 open for userB, 1 won for userB (must NOT count).
	seedLeadForUser(t, db, fxA, userA, domain.LeadStatusOpen)
	seedLeadForUser(t, db, fxA, userA, domain.LeadStatusOpen)
	seedLeadForUser(t, db, fxA, userA, domain.LeadStatusOpen)
	seedLeadForUser(t, db, fxA, userB, domain.LeadStatusOpen)
	seedLeadForUser(t, db, fxA, userB, domain.LeadStatusWon)

	// Tenant B: 1 open for userA — must NOT leak across tenants.
	seedLeadForUser(t, db, fxB, userA, domain.LeadStatusOpen)

	got, err := repos.leads.CountActiveByUsers(ctx, fxA.tenantID, []string{userA, userB, userC})
	require.NoError(t, err)

	// userA: 3 open in tenant A; userB: 1 open (won excluded); userC absent (zero results).
	assert.Equal(t, map[string]int{userA: 3, userB: 1}, got)
	_, hasC := got[userC]
	assert.False(t, hasC, "userC must be absent from the map (zero open leads)")
}
