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
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil)

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
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil)

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

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil)

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

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil)

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
	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil)

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

	uc := NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, detector, router, nil)

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
