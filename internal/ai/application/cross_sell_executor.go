package application

import (
	"context"
	"fmt"
	"strings"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ProductSpecialistResolver finds the specialist responsible for a given product.
type ProductSpecialistResolver interface {
	FindSpecialistByProduct(ctx context.Context, productID string) (specialistID, funnelID, initialColumnID string, err error)
}

// LeadFactory creates a new lead as part of a cross-sell transition.
type LeadFactory interface {
	CreateForCrossSell(ctx context.Context, originLeadID, tenantID, funnelID, columnID, specialistID string) (newLeadID string, err error)
}

// ConversationMover handles specialist migration and pending cross-sell state on a conversation.
type ConversationMover interface {
	MigrateSpecialist(ctx context.Context, conversationID, newSpecialistID string) error
	SetPendingCrossSell(ctx context.Context, conversationID, ruleID string) error
	ClearPendingCrossSell(ctx context.Context, conversationID string) error
}

// ProductNameLookup retrieves a human-readable product name by ID.
type ProductNameLookup interface {
	Name(ctx context.Context, productID string) (string, error)
}

// CrossSellExecutor decides the initial action when a cross-sell rule fires and
// later completes the specialist transition when appropriate.
type CrossSellExecutor struct {
	productResolver   ProductSpecialistResolver
	leadFactory       LeadFactory
	conversationMover ConversationMover
	leadUpdater       LeadUpdater
	sender            MessageSender
	productNameLookup ProductNameLookup
}

// NewCrossSellExecutor constructs a CrossSellExecutor with all required dependencies.
func NewCrossSellExecutor(
	pr ProductSpecialistResolver,
	lf LeadFactory,
	cm ConversationMover,
	lu LeadUpdater,
	ms MessageSender,
	pnl ProductNameLookup,
) *CrossSellExecutor {
	return &CrossSellExecutor{
		productResolver:   pr,
		leadFactory:       lf,
		conversationMover: cm,
		leadUpdater:       lu,
		sender:            ms,
		productNameLookup: pnl,
	}
}

// Execute decides the initial action based on the specialist's CrossSellMode.
//
//   - confirm  → sends a confirmation question, marks conversation with pending rule; NO transition yet.
//   - announce → renders and sends the announcement template, then calls CompleteTransition.
//   - silent   → no message sent, calls CompleteTransition directly.
//
// CompleteTransition must be called separately by the caller after a positive
// confirmation answer when mode == confirm.
func (x *CrossSellExecutor) Execute(
	ctx context.Context,
	convID, tenantID, originLeadID, crossSellColumnID string,
	fromSpecialist *specDomain.Specialist,
	rule *specDomain.CrossSellRule,
) error {
	productName, err := x.productNameLookup.Name(ctx, rule.TargetProductID)
	if err != nil {
		return fmt.Errorf("cross_sell_executor: lookup product name: %w", err)
	}

	switch fromSpecialist.CrossSellMode {
	case specDomain.CrossSellModeConfirm:
		msg := fmt.Sprintf("Posso te conectar com nosso especialista em %s?", productName)
		if err := x.sender.SendAIResponse(ctx, tenantID, convID, msg); err != nil {
			return fmt.Errorf("cross_sell_executor: send confirm question: %w", err)
		}
		if err := x.conversationMover.SetPendingCrossSell(ctx, convID, rule.ID); err != nil {
			return fmt.Errorf("cross_sell_executor: set pending cross-sell: %w", err)
		}
		return nil

	case specDomain.CrossSellModeAnnounce:
		rendered := strings.ReplaceAll(fromSpecialist.CrossSellAnnouncementTemplate, "{{produto}}", productName)
		if err := x.sender.SendAIResponse(ctx, tenantID, convID, rendered); err != nil {
			return fmt.Errorf("cross_sell_executor: send announcement: %w", err)
		}

	case specDomain.CrossSellModeSilent:
		// no message

	default:
		return fmt.Errorf("cross_sell_executor: unknown cross-sell mode %q", fromSpecialist.CrossSellMode)
	}

	return x.CompleteTransition(ctx, convID, tenantID, originLeadID, crossSellColumnID, rule)
}

// CompleteTransition performs the full specialist handoff:
//  1. Resolves new specialist via target product.
//  2. Creates a new lead linked to the origin lead.
//  3. Marks current lead outcome as cross_sell.
//  4. Moves current lead to crossSellColumnID (if provided).
//  5. Migrates the conversation to the new specialist.
//
// This is called immediately for announce/silent modes, or by the confirm handler
// after a positive user reply.
func (x *CrossSellExecutor) CompleteTransition(
	ctx context.Context,
	convID, tenantID, originLeadID, crossSellColumnID string,
	rule *specDomain.CrossSellRule,
) error {
	newSpecID, funnelID, colID, err := x.productResolver.FindSpecialistByProduct(ctx, rule.TargetProductID)
	if err != nil {
		return fmt.Errorf("cross_sell_executor: find specialist by product: %w", err)
	}

	if _, err := x.leadFactory.CreateForCrossSell(ctx, originLeadID, tenantID, funnelID, colID, newSpecID); err != nil {
		return fmt.Errorf("cross_sell_executor: create cross-sell lead: %w", err)
	}

	if err := x.leadUpdater.SetOutcome(ctx, convID, string(specDomain.OutcomeCrossSell)); err != nil {
		return fmt.Errorf("cross_sell_executor: set outcome: %w", err)
	}

	if crossSellColumnID != "" {
		// best-effort: move origin lead to the configured column
		_ = x.leadUpdater.MoveLeadToColumn(ctx, convID, crossSellColumnID)
	}

	if err := x.conversationMover.MigrateSpecialist(ctx, convID, newSpecID); err != nil {
		return fmt.Errorf("cross_sell_executor: migrate specialist: %w", err)
	}

	return nil
}
