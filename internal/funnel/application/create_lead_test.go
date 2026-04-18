package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
)

// fakePicker is the default picker used across tests. Unless a sub-test
// overrides it, it returns a fixed "default-responsible" user so the existing
// CreateLead flow stays green after the picker became mandatory.
type fakePicker struct {
	result   domain.PickResult
	err      error
	lastCall struct{ tenant, funnel, col string }
}

func (f *fakePicker) PickForFunnel(_ context.Context, tenantID, funnelID, columnID string) (domain.PickResult, error) {
	f.lastCall.tenant, f.lastCall.funnel, f.lastCall.col = tenantID, funnelID, columnID
	return f.result, f.err
}

func defaultFakePicker() *fakePicker {
	return &fakePicker{result: domain.PickResult{UserID: "default-responsible", Outcome: domain.PickOutcomePicked}}
}

// fakeEventBus captures published events for assertion. Subscribe is a no-op
// because CreateLead does not consume it.
type fakeEventBus struct {
	events []events.Event
}

func (b *fakeEventBus) Publish(e events.Event) { b.events = append(b.events, e) }

func (b *fakeEventBus) Subscribe(_ string) (<-chan events.Event, func()) {
	ch := make(chan events.Event)
	close(ch)
	return ch, func() {}
}

func setupCreateLeadTest(t *testing.T) (*CreateLeadUseCase, *mockFunnelRepo, *mockColumnRepo, *mockLeadRepo, *mockLeadMovementRepo, *domain.Funnel, *domain.Column) {
	t.Helper()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil, defaultFakePicker())

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
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID: "tenant-1", ContactID: "contact-1", ConversationID: "conv-1",
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestCreateLead_CreateFromConversation(t *testing.T) {
	uc, _, _, leadRepo, _, _, _ := setupCreateLeadTest(t)

	err := uc.CreateFromConversation(context.Background(), "tenant-1", "contact-1", "conv-1", "")
	require.NoError(t, err)
	assert.Len(t, leadRepo.leads, 1)
}

func TestCreateLead_ProductDetected_RoutesToCorrectFunnel(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()

	// Default funnel
	defaultFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Default Pipeline", "")
	defaultFunnel.SetDefault()
	_ = funnelRepo.Create(context.Background(), defaultFunnel)
	defaultEntry, _ := domain.NewColumn(uuid.New().String(), defaultFunnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), defaultEntry)

	// Product-specific funnel
	productFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Divorcio Pipeline", "")
	_ = funnelRepo.Create(context.Background(), productFunnel)
	productEntry, _ := domain.NewColumn(uuid.New().String(), productFunnel.ID, "Entrada", 0, domain.ColumnTypeEntry, "#3b82f6")
	_ = columnRepo.Create(context.Background(), productEntry)

	// Setup product detection
	detector := newMockProductDetector()
	detector.AddResult("tenant-1", "divorcio", "product-1")

	router := newMockFunnelProductRouter()
	router.routes["product-1"] = productFunnel.ID

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		MessageText:    "Preciso de ajuda com divorcio",
	})

	require.NoError(t, err)
	require.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, productFunnel.ID, l.FunnelID, "lead should be routed to product-specific funnel")
		assert.Equal(t, productEntry.ID, l.ColumnID, "lead should be in product funnel entry column")
		assert.Equal(t, "product-1", l.ProductID, "lead should have product ID set")
	}
}

func TestCreateLead_ProductRoutesToOtherTenantFunnel_IsIgnored(t *testing.T) {
	// Regression: the router used to query funnel_products by product_id only,
	// which could return a funnel belonging to another tenant. The lead would
	// then be created under a foreign tenant's funnel and vanish from the
	// caller tenant's kanban. See gorm_funnel_product_repository.go.
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()

	// Callers tenant: default funnel only.
	defaultFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Default", "")
	defaultFunnel.SetDefault()
	_ = funnelRepo.Create(context.Background(), defaultFunnel)
	defaultEntry, _ := domain.NewColumn(uuid.New().String(), defaultFunnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), defaultEntry)

	// Product detection hits, but the only route registered is for ANOTHER tenant.
	detector := newMockProductDetector()
	detector.AddResult("tenant-1", "foo", "product-1")

	router := newMockFunnelProductRouter()
	router.routes["tenant-2|product-1"] = "funnel-of-tenant-2" // cross-tenant; must be ignored

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		MessageText:    "foo",
	})

	require.NoError(t, err)
	require.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, defaultFunnel.ID, l.FunnelID, "must fall back to own-tenant default funnel, never route to another tenant's funnel")
	}
}

func TestCreateLead_NoProductDetected_DefaultFunnel(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()

	// Default funnel
	defaultFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Default Pipeline", "")
	defaultFunnel.SetDefault()
	_ = funnelRepo.Create(context.Background(), defaultFunnel)
	defaultEntry, _ := domain.NewColumn(uuid.New().String(), defaultFunnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), defaultEntry)

	// Setup product detection (no keywords match)
	detector := newMockProductDetector()
	router := newMockFunnelProductRouter()

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		MessageText:    "Ola, preciso de ajuda",
	})

	require.NoError(t, err)
	require.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, defaultFunnel.ID, l.FunnelID, "lead should use default funnel")
		assert.Equal(t, defaultEntry.ID, l.ColumnID, "lead should be in default funnel entry column")
		assert.Empty(t, l.ProductID, "lead should not have product ID")
	}
}

func TestCreateLead_NilDetector_DefaultFunnel(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()

	// Default funnel
	defaultFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Default Pipeline", "")
	defaultFunnel.SetDefault()
	_ = funnelRepo.Create(context.Background(), defaultFunnel)
	defaultEntry, _ := domain.NewColumn(uuid.New().String(), defaultFunnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), defaultEntry)

	// nil detector and router (backward compatible)
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		MessageText:    "Preciso de ajuda com divorcio",
	})

	require.NoError(t, err)
	require.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, defaultFunnel.ID, l.FunnelID, "lead should use default funnel when detector is nil")
		assert.Empty(t, l.ProductID, "lead should not have product ID when detector is nil")
	}
}

func TestCreateLead_ProductDetected_InactiveFunnel_FallsBack(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()

	// Default funnel
	defaultFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Default Pipeline", "")
	defaultFunnel.SetDefault()
	_ = funnelRepo.Create(context.Background(), defaultFunnel)
	defaultEntry, _ := domain.NewColumn(uuid.New().String(), defaultFunnel.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), defaultEntry)

	// Inactive product-specific funnel
	productFunnel, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Divorcio Pipeline", "")
	productFunnel.Deactivate()
	_ = funnelRepo.Create(context.Background(), productFunnel)

	detector := newMockProductDetector()
	detector.AddResult("tenant-1", "divorcio", "product-1")

	router := newMockFunnelProductRouter()
	router.routes["product-1"] = productFunnel.ID

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil, defaultFakePicker())

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
		MessageText:    "Preciso de ajuda com divorcio",
	})

	require.NoError(t, err)
	require.Len(t, leadRepo.leads, 1)
	for _, l := range leadRepo.leads {
		assert.Equal(t, defaultFunnel.ID, l.FunnelID, "should fall back to default when product funnel is inactive")
		assert.Equal(t, "product-1", l.ProductID, "product should still be detected even if funnel is inactive")
	}
}

// --- Task 11: ResponsiblePicker integration tests ---

func TestCreateLeadUseCase_AssignsResponsibleFromPicker(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	bus := &fakeEventBus{}
	picker := &fakePicker{result: domain.PickResult{
		UserID:    "user-99",
		Algorithm: "round_robin",
		GroupID:   "g1",
		Outcome:   domain.PickOutcomePicked,
	}}

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.SetDefault()
	_ = funnelRepo.Create(context.Background(), f)
	entryCol, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), entryCol)

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, bus, picker)

	input := CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
	}
	err := uc.Execute(context.Background(), input)
	require.NoError(t, err)

	// Lead got the responsible.
	require.Len(t, leadRepo.leads, 1)
	var created *domain.Lead
	for _, l := range leadRepo.leads {
		created = l
	}
	require.NotNil(t, created)
	assert.Equal(t, "user-99", created.ResponsibleUserID)

	// Picker was invoked with the right scope.
	assert.Equal(t, input.TenantID, picker.lastCall.tenant)
	assert.Equal(t, f.ID, picker.lastCall.funnel)
	assert.Equal(t, entryCol.ID, picker.lastCall.col)

	// Events: EventLeadCreated FIRST, then EventLeadResponsibleAssigned.
	require.Len(t, bus.events, 2)
	assert.Equal(t, events.EventLeadCreated, bus.events[0].Type)
	createdPayload, ok := bus.events[0].Payload.(map[string]string)
	require.True(t, ok, "EventLeadCreated payload should be map[string]string")
	assert.Equal(t, "user-99", createdPayload["responsible_user_id"])
	assert.Equal(t, created.ID, createdPayload["lead_id"])

	assert.Equal(t, events.EventLeadResponsibleAssigned, bus.events[1].Type)
	assignedPayload, ok := bus.events[1].Payload.(events.ResponsibleAssignedPayload)
	require.True(t, ok, "EventLeadResponsibleAssigned payload should be ResponsibleAssignedPayload")
	assert.Equal(t, created.ID, assignedPayload.LeadID)
	assert.Equal(t, "user-99", assignedPayload.ResponsibleUserID)
	assert.Equal(t, "created", assignedPayload.Reason)
	assert.Equal(t, "picked", assignedPayload.Outcome)
	assert.Equal(t, "round_robin", assignedPayload.Algorithm)
}

func TestCreateLeadUseCase_AbortsWhenPickerReturnsHardError(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	movementRepo := newMockLeadMovementRepo()
	bus := &fakeEventBus{}
	picker := &fakePicker{err: domain.ErrNoResponsibleAvailable}

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.SetDefault()
	_ = funnelRepo.Create(context.Background(), f)
	entryCol, _ := domain.NewColumn(uuid.New().String(), f.ID, "Novo", 0, domain.ColumnTypeEntry, "#22c55e")
	_ = columnRepo.Create(context.Background(), entryCol)

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, bus, picker)

	err := uc.Execute(context.Background(), CreateLeadInput{
		TenantID:       "tenant-1",
		ContactID:      "contact-1",
		ConversationID: "conv-1",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNoResponsibleAvailable), "error must wrap ErrNoResponsibleAvailable")

	// Lead was not persisted.
	assert.Equal(t, 0, len(leadRepo.leads), "lead must not be created when picker fails hard")
	assert.Equal(t, 0, len(movementRepo.movements), "movement must not be created when picker fails hard")

	// No events published.
	for _, ev := range bus.events {
		assert.NotEqual(t, events.EventLeadCreated, ev.Type, "EventLeadCreated must not be published on picker failure")
		assert.NotEqual(t, events.EventLeadResponsibleAssigned, ev.Type, "EventLeadResponsibleAssigned must not be published on picker failure")
	}
}
