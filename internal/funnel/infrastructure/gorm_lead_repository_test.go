package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestGormLead_PersistsOutcomeFields(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()

	fx := newLeadFixture(t, repos, db)

	// Create a second lead to act as cross_sell origin (satisfies self-referencing FK).
	originContactID := createContact(t, db, fx.tenantID, "Origem", "+5511988880001")
	originConvID := createConversation(t, db, fx.tenantID, originContactID)
	originLead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, originContactID, originConvID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, originLead))

	// Create the lead under test with outcome fields set.
	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	lead.QualificationOutcome = domain.QualificationOutcomeHumano
	originID := originLead.ID
	lead.CrossSellOriginLeadID = &originID

	require.NoError(t, repos.leads.Create(ctx, lead))

	got, err := repos.leads.FindByID(ctx, lead.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QualificationOutcomeHumano, got.QualificationOutcome)
	require.NotNil(t, got.CrossSellOriginLeadID)
	assert.Equal(t, originLead.ID, *got.CrossSellOriginLeadID)
}

func TestGormLead_DefaultOutcomeIsEmAndamento(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()

	fx := newLeadFixture(t, repos, db)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	got, err := repos.leads.FindByID(ctx, lead.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QualificationOutcomeEmAndamento, got.QualificationOutcome)
	assert.Nil(t, got.CrossSellOriginLeadID)
}

func TestGormLead_UpdateOutcomeFields(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()

	fx := newLeadFixture(t, repos, db)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	// Update outcome.
	lead.SetQualificationOutcome(domain.QualificationOutcomeAprovado)
	require.NoError(t, repos.leads.Update(ctx, lead))

	got, err := repos.leads.FindByID(ctx, lead.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.QualificationOutcomeAprovado, got.QualificationOutcome)
}
