package infrastructure

import (
	"context"
	"testing"
	"time"

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

func TestGormLead_FindCurrentByConversationID_ReturnsMostRecent(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()
	fx := newLeadFixture(t, repos, db)

	// Lead antigo na conversa.
	old, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	old.CreatedAt = time.Now().Add(-1 * time.Hour)
	require.NoError(t, repos.leads.Create(ctx, old))

	// Lead novo (destino de cross-sell) na MESMA conversa.
	recent, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	recent.CreatedAt = time.Now()
	require.NoError(t, repos.leads.Create(ctx, recent))

	got, err := repos.leads.FindCurrentByConversationID(ctx, fx.tenantID, fx.conversationID)
	require.NoError(t, err)
	assert.Equal(t, recent.ID, got.ID)
}

func TestGormLead_FindCurrentByConversationID_TenantIsolation(t *testing.T) {
	repos, db := setupFunnelRepos(t)
	ctx := context.Background()
	fx := newLeadFixture(t, repos, db)

	lead, err := domain.NewLead(uuid.New().String(), fx.tenantID, fx.funnel.ID, fx.entryColumn.ID, fx.contactID, fx.conversationID)
	require.NoError(t, err)
	require.NoError(t, repos.leads.Create(ctx, lead))

	_, err = repos.leads.FindCurrentByConversationID(ctx, "outro-tenant", fx.conversationID)
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestGormLead_FindCurrentByConversationID_NotFound(t *testing.T) {
	repos, _ := setupFunnelRepos(t)
	_, err := repos.leads.FindCurrentByConversationID(context.Background(), "t", uuid.New().String())
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
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
