package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	aiDomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
)

// ─── ProductSpecialistResolver ────────────────────────────────────────────────

// ProductSpecialistResolverAdapter resolves the specialist, funnel, and initial column
// for a given product. It combines:
//   - ai/domain.SpecialistProductRepository  (productID → specialistID)
//   - product/domain.FunnelProductRepository  (tenantID + productID → funnelID)
//   - funnel/domain.ColumnRepository          (funnelID → entry column)
type ProductSpecialistResolverAdapter struct {
	spProductRepo       aiDomain.SpecialistProductRepository
	funnelProductFinder funnelProductTopPriorityFinder
	columnEntryFinder   funnelEntryColumnFinder
}

// funnelProductTopPriorityFinder is a minimal interface covering the single method we need.
type funnelProductTopPriorityFinder interface {
	FindTopPriorityFunnel(ctx context.Context, tenantID, productID string) (*productDomain.FunnelProduct, error)
}

// funnelEntryColumnFinder is a minimal interface covering the single method we need.
type funnelEntryColumnFinder interface {
	FindEntryByFunnelID(ctx context.Context, funnelID string) (*funnelDomain.Column, error)
}

// NewProductSpecialistResolverAdapter builds a ProductSpecialistResolverAdapter.
// tenantID is required so FindTopPriorityFunnel can scope by tenant.
func NewProductSpecialistResolverAdapter(
	spProductRepo aiDomain.SpecialistProductRepository,
	funnelProductFinder funnelProductTopPriorityFinder,
	columnEntryFinder funnelEntryColumnFinder,
) *ProductSpecialistResolverAdapter {
	return &ProductSpecialistResolverAdapter{
		spProductRepo:       spProductRepo,
		funnelProductFinder: funnelProductFinder,
		columnEntryFinder:   columnEntryFinder,
	}
}

// FindSpecialistByProduct returns (specialistID, funnelID, initialColumnID) for the given productID.
// The tenant scope must be embedded in funnelProductFinder — callers wire a tenant-scoped version
// or the repo accepts tenantID via context value if needed.
// NOTE: tenantID is passed as an empty string to FindTopPriorityFunnel when the engine doesn't
// have it; callers should ensure a tenant-scoped adapter when multi-tenant isolation matters.
func (a *ProductSpecialistResolverAdapter) FindSpecialistByProduct(ctx context.Context, productID string) (specialistID, funnelID, initialColumnID string, err error) {
	// 1. Resolve specialist.
	sps, err := a.spProductRepo.FindByProductID(ctx, productID)
	if err != nil {
		return "", "", "", fmt.Errorf("product_specialist_resolver: find by product: %w", err)
	}
	if len(sps) == 0 {
		return "", "", "", fmt.Errorf("product_specialist_resolver: no specialist linked to product %s", productID)
	}
	specialistID = sps[0].SpecialistID

	// 2. Resolve funnel (tenant not needed since repo is already scoped, pass empty to satisfy signature).
	fp, err := a.funnelProductFinder.FindTopPriorityFunnel(ctx, "", productID)
	if err != nil {
		return "", "", "", fmt.Errorf("product_specialist_resolver: find funnel for product: %w", err)
	}
	funnelID = fp.FunnelID

	// 3. Resolve entry column.
	col, err := a.columnEntryFinder.FindEntryByFunnelID(ctx, funnelID)
	if err != nil {
		return "", "", "", fmt.Errorf("product_specialist_resolver: find entry column: %w", err)
	}
	initialColumnID = col.ID

	return specialistID, funnelID, initialColumnID, nil
}

// ─── LeadFactory ─────────────────────────────────────────────────────────────

// LeadFactoryAdapter creates a new lead as part of a cross-sell transition.
// It uses the existing lead to obtain contactID so the new lead is linked to the same contact.
type LeadFactoryAdapter struct {
	leadRepo funnelDomain.LeadRepository
}

// NewLeadFactoryAdapter builds a LeadFactoryAdapter.
func NewLeadFactoryAdapter(leadRepo funnelDomain.LeadRepository) *LeadFactoryAdapter {
	return &LeadFactoryAdapter{leadRepo: leadRepo}
}

// CreateForCrossSell looks up the origin lead by its lead ID, then creates a new
// lead in the target funnel/column with CrossSellOriginLeadID set to the origin lead's ID.
// originLeadID must be the actual lead.ID (not a conversation ID).
func (a *LeadFactoryAdapter) CreateForCrossSell(
	ctx context.Context,
	originLeadID, tenantID, funnelID, columnID, specialistID string,
) (newLeadID string, err error) {
	// Resolve origin lead by its primary key to get contactID and conversationID.
	originLead, err := a.leadRepo.FindByID(ctx, originLeadID)
	if err != nil {
		return "", fmt.Errorf("lead_factory: find origin lead: %w", err)
	}

	newLeadID = uuid.New().String()
	newLead, err := funnelDomain.NewLead(newLeadID, tenantID, funnelID, columnID, originLead.ContactID, originLead.ConversationID)
	if err != nil {
		return "", fmt.Errorf("lead_factory: construct new lead: %w", err)
	}
	newLead.CrossSellOriginLeadID = &originLeadID

	if err := a.leadRepo.Create(ctx, newLead); err != nil {
		return "", fmt.Errorf("lead_factory: create lead: %w", err)
	}

	return newLeadID, nil
}

// ─── ConversationMover ───────────────────────────────────────────────────────

// ConversationMoverAdapter implements ConversationMover using:
//   - ai/domain.ConversationStateRepository  (for PendingCrossSellRuleID and SpecialistID update)
type ConversationMoverAdapter struct {
	stateRepo aiDomain.ConversationStateRepository
}

// NewConversationMoverAdapter builds a ConversationMoverAdapter.
func NewConversationMoverAdapter(stateRepo aiDomain.ConversationStateRepository) *ConversationMoverAdapter {
	return &ConversationMoverAdapter{stateRepo: stateRepo}
}

// MigrateSpecialist updates the ConversationState.SpecialistID to the new specialist,
// effectively routing future messages to the new specialist.
func (a *ConversationMoverAdapter) MigrateSpecialist(ctx context.Context, conversationID, newSpecialistID string) error {
	state, err := a.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("conversation_mover: find state: %w", err)
	}
	state.SpecialistID = newSpecialistID
	if err := a.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("conversation_mover: migrate specialist: %w", err)
	}
	return nil
}

// SetPendingCrossSell sets PendingCrossSellRuleID on the ConversationState.
func (a *ConversationMoverAdapter) SetPendingCrossSell(ctx context.Context, conversationID, ruleID string) error {
	state, err := a.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("conversation_mover: find state for set-pending: %w", err)
	}
	state.SetPendingCrossSellRuleID(ruleID)
	if err := a.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("conversation_mover: set pending cross-sell: %w", err)
	}
	return nil
}

// ClearPendingCrossSell removes the pending cross-sell rule ID from the ConversationState.
func (a *ConversationMoverAdapter) ClearPendingCrossSell(ctx context.Context, conversationID string) error {
	state, err := a.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if errors.Is(err, aiDomain.ErrConversationStateNotFound) {
			return nil // already cleared or never set
		}
		return fmt.Errorf("conversation_mover: find state for clear-pending: %w", err)
	}
	state.ClearPendingCrossSellRuleID()
	if err := a.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("conversation_mover: clear pending cross-sell: %w", err)
	}
	return nil
}

// ─── ProductNameLookup ────────────────────────────────────────────────────────

// ProductNameLookupAdapter resolves a human-readable product name by ID.
type ProductNameLookupAdapter struct {
	productRepo productDomain.ProductRepository
}

// NewProductNameLookupAdapter builds a ProductNameLookupAdapter.
func NewProductNameLookupAdapter(productRepo productDomain.ProductRepository) *ProductNameLookupAdapter {
	return &ProductNameLookupAdapter{productRepo: productRepo}
}

// Name returns the product's name for the given productID.
func (a *ProductNameLookupAdapter) Name(ctx context.Context, productID string) (string, error) {
	product, err := a.productRepo.FindByID(ctx, productID)
	if err != nil {
		return "", fmt.Errorf("product_name_lookup: find product: %w", err)
	}
	return product.Name, nil
}

// ─── gorm FunnelProductRepository shim ──────────────────────────────────────
// The FunnelProductRepository is in the product package but we need to implement
// funnelProductTopPriorityFinder. We create a thin wrapper around the GORM repo.

// GormFunnelProductFinder wraps a GORM DB to satisfy funnelProductTopPriorityFinder.
// This thin shim is used by NewProductSpecialistResolverAdapter when a full
// product/infrastructure.GormFunnelProductRepository is not yet injected.
// TODO(B9): replace with the concrete product/infrastructure.GormFunnelProductRepository
// once it is wired via ModuleDeps.
type GormFunnelProductFinder struct {
	db *gorm.DB
}

// NewGormFunnelProductFinder creates a GormFunnelProductFinder.
func NewGormFunnelProductFinder(db *gorm.DB) *GormFunnelProductFinder {
	return &GormFunnelProductFinder{db: db}
}

// funnelProductModel is a GORM model for the funnel_products table.
type funnelProductModel struct {
	ID        string `gorm:"primaryKey;column:id"`
	FunnelID  string `gorm:"column:funnel_id"`
	ProductID string `gorm:"column:product_id"`
	Priority  int    `gorm:"column:priority"`
}

func (funnelProductModel) TableName() string { return "funnel_products" }

// FindTopPriorityFunnel returns the highest-priority funnel-product association.
func (f *GormFunnelProductFinder) FindTopPriorityFunnel(ctx context.Context, _ /*tenantID*/ string, productID string) (*productDomain.FunnelProduct, error) {
	var m funnelProductModel
	if err := f.db.WithContext(ctx).
		Where("product_id = ?", productID).
		Order("priority DESC").
		First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("funnel_product: no funnel for product %s", productID)
		}
		return nil, fmt.Errorf("funnel_product: query: %w", err)
	}
	fp, err := productDomain.NewFunnelProduct(m.ID, m.FunnelID, m.ProductID, m.Priority)
	if err != nil {
		return nil, fmt.Errorf("funnel_product: reconstruct domain: %w", err)
	}
	return fp, nil
}
