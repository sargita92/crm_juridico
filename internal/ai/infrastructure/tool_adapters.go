package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	aiDomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/ai/infrastructure/tools"
	automationApp "github.com/sasrgita/crm-juridico/internal/automation/application"
	funnelApp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funnelDomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	productDomain "github.com/sasrgita/crm-juridico/internal/product/domain"
	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// ---------------------------------------------------------------------------
// 1. LeadSearchAdapter — satisfies tools.LeadSearcher
// ---------------------------------------------------------------------------

// LeadSearchAdapter wraps LeadRepository to support cross-funnel tenant search.
type LeadSearchAdapter struct {
	repo funnelDomain.LeadRepository
}

// NewLeadSearchAdapter creates a LeadSearchAdapter.
func NewLeadSearchAdapter(repo funnelDomain.LeadRepository) *LeadSearchAdapter {
	return &LeadSearchAdapter{repo: repo}
}

// SearchByTenant searches leads by contact name, phone, or status within a tenant.
func (a *LeadSearchAdapter) SearchByTenant(ctx context.Context, tenantID, query string) ([]funnelDomain.Lead, error) {
	leads, err := a.repo.FindByTenantAndSearch(ctx, tenantID, query, 20)
	if err != nil {
		return nil, fmt.Errorf("lead_search_adapter: search by tenant: %w", err)
	}
	return leads, nil
}

// ---------------------------------------------------------------------------
// 2. LeadDetailAdapter — satisfies tools.LeadDetailFetcher
// ---------------------------------------------------------------------------

// LeadDetailAdapter wraps LeadRepository and LeadNoteRepository.
type LeadDetailAdapter struct {
	leadRepo funnelDomain.LeadRepository
	noteRepo funnelDomain.LeadNoteRepository
}

// NewLeadDetailAdapter creates a LeadDetailAdapter.
func NewLeadDetailAdapter(leadRepo funnelDomain.LeadRepository, noteRepo funnelDomain.LeadNoteRepository) *LeadDetailAdapter {
	return &LeadDetailAdapter{leadRepo: leadRepo, noteRepo: noteRepo}
}

// FindByID returns the lead only if it belongs to the given tenant.
func (a *LeadDetailAdapter) FindByID(ctx context.Context, tenantID, leadID string) (*funnelDomain.Lead, error) {
	lead, err := a.leadRepo.FindByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, funnelDomain.ErrLeadNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("lead_detail_adapter: find by id: %w", err)
	}
	if lead.TenantID != tenantID {
		return nil, nil // not found for this tenant
	}
	return lead, nil
}

// FindNotes returns the notes for a lead, verifying tenant ownership first.
func (a *LeadDetailAdapter) FindNotes(ctx context.Context, tenantID, leadID string) ([]funnelDomain.LeadNote, error) {
	// verify ownership
	lead, err := a.FindByID(ctx, tenantID, leadID)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, nil
	}
	notes, err := a.noteRepo.FindByLeadID(ctx, leadID)
	if err != nil {
		return nil, fmt.Errorf("lead_detail_adapter: find notes: %w", err)
	}
	return notes, nil
}

// ---------------------------------------------------------------------------
// 3. ConversationHistoryToolAdapter — satisfies tools.ConversationHistoryFetcher
// ---------------------------------------------------------------------------

// ConversationHistoryToolAdapter bridges the whatsapp MessageRepository to the
// tools.ConversationHistoryFetcher interface using the self-contained HistoryMessage type.
type ConversationHistoryToolAdapter struct {
	repo whatsappDomain.MessageRepository
}

// NewConversationHistoryToolAdapter creates a ConversationHistoryToolAdapter.
func NewConversationHistoryToolAdapter(repo whatsappDomain.MessageRepository) *ConversationHistoryToolAdapter {
	return &ConversationHistoryToolAdapter{repo: repo}
}

// FindMessages retrieves recent messages and maps direction to role.
func (a *ConversationHistoryToolAdapter) FindMessages(ctx context.Context, conversationID string, limit int) ([]tools.HistoryMessage, error) {
	messages, err := a.repo.FindByConversationID(ctx, conversationID, whatsappDomain.MessageFilter{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("conv_history_tool_adapter: find by conversation: %w", err)
	}

	result := make([]tools.HistoryMessage, len(messages))
	for i, m := range messages {
		role := "user"
		if m.Direction == whatsappDomain.MessageDirectionOutgoing {
			role = "assistant"
		}
		result[i] = tools.HistoryMessage{
			Role:      role,
			Content:   m.Content,
			Timestamp: m.Timestamp,
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 4. ProductListToolAdapter — satisfies tools.ProductLister
// ---------------------------------------------------------------------------

// ProductListToolAdapter wraps TenantProductRepository + ProductRepository to
// list the products active for a specific tenant.
type ProductListToolAdapter struct {
	tenantProductRepo productDomain.TenantProductRepository
	productRepo       productDomain.ProductRepository
}

// NewProductListToolAdapter creates a ProductListToolAdapter.
func NewProductListToolAdapter(
	tenantProductRepo productDomain.TenantProductRepository,
	productRepo productDomain.ProductRepository,
) *ProductListToolAdapter {
	return &ProductListToolAdapter{
		tenantProductRepo: tenantProductRepo,
		productRepo:       productRepo,
	}
}

// ListByTenant returns the active products linked to this tenant.
func (a *ProductListToolAdapter) ListByTenant(ctx context.Context, tenantID string) ([]productDomain.Product, error) {
	tenantProducts, err := a.tenantProductRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("product_list_tool_adapter: find tenant products: %w", err)
	}
	if len(tenantProducts) == 0 {
		return nil, nil
	}

	ids := make([]string, len(tenantProducts))
	for i, tp := range tenantProducts {
		ids[i] = tp.ProductID
	}

	products, err := a.productRepo.FindActiveByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("product_list_tool_adapter: find active products: %w", err)
	}
	return products, nil
}

// ---------------------------------------------------------------------------
// 5. PipelineToolAdapter — satisfies tools.PipelineFetcher
// ---------------------------------------------------------------------------

// PipelineToolAdapter wraps FunnelRepository, ColumnRepository, and LeadRepository.
type PipelineToolAdapter struct {
	funnelRepo funnelDomain.FunnelRepository
	colRepo    funnelDomain.ColumnRepository
	leadRepo   funnelDomain.LeadRepository
}

// NewPipelineToolAdapter creates a PipelineToolAdapter.
func NewPipelineToolAdapter(
	funnelRepo funnelDomain.FunnelRepository,
	colRepo funnelDomain.ColumnRepository,
	leadRepo funnelDomain.LeadRepository,
) *PipelineToolAdapter {
	return &PipelineToolAdapter{funnelRepo: funnelRepo, colRepo: colRepo, leadRepo: leadRepo}
}

// GetPipeline returns the column summaries for the given funnel (or default funnel if funnelID is empty).
func (a *PipelineToolAdapter) GetPipeline(ctx context.Context, tenantID, funnelID string) ([]tools.ColumnSummary, error) {
	var targetFunnelID string

	if funnelID == "" {
		funnel, err := a.funnelRepo.FindDefaultByTenantID(ctx, tenantID)
		if err != nil {
			if errors.Is(err, funnelDomain.ErrFunnelNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("pipeline_tool_adapter: find default funnel: %w", err)
		}
		if funnel == nil {
			return nil, nil
		}
		targetFunnelID = funnel.ID
	} else {
		// validate the funnel belongs to the tenant
		funnel, err := a.funnelRepo.FindByID(ctx, funnelID)
		if err != nil {
			return nil, fmt.Errorf("pipeline_tool_adapter: find funnel: %w", err)
		}
		if funnel.TenantID != tenantID {
			return nil, fmt.Errorf("pipeline_tool_adapter: funnel %s not found for tenant", funnelID)
		}
		targetFunnelID = funnel.ID
	}

	cols, err := a.colRepo.FindByFunnelID(ctx, targetFunnelID)
	if err != nil {
		return nil, fmt.Errorf("pipeline_tool_adapter: find columns: %w", err)
	}

	summaries := make([]tools.ColumnSummary, 0, len(cols))
	for _, col := range cols {
		count, err := a.leadRepo.CountByColumnID(ctx, col.ID)
		if err != nil {
			return nil, fmt.Errorf("pipeline_tool_adapter: count leads in column %s: %w", col.ID, err)
		}
		summaries = append(summaries, tools.ColumnSummary{
			ID:        col.ID,
			Name:      col.Name,
			LeadCount: count,
		})
	}
	return summaries, nil
}

// ---------------------------------------------------------------------------
// 6. LeadMoveToolAdapter — satisfies tools.LeadMover
// ---------------------------------------------------------------------------

// LeadMoveToolAdapter wraps MoveLeadUseCase with tenant verification.
type LeadMoveToolAdapter struct {
	leadRepo   funnelDomain.LeadRepository
	moveLeadUC *funnelApp.MoveLeadUseCase
}

// NewLeadMoveToolAdapter creates a LeadMoveToolAdapter.
func NewLeadMoveToolAdapter(leadRepo funnelDomain.LeadRepository, moveLeadUC *funnelApp.MoveLeadUseCase) *LeadMoveToolAdapter {
	return &LeadMoveToolAdapter{leadRepo: leadRepo, moveLeadUC: moveLeadUC}
}

// MoveToColumn verifies tenant ownership and moves the lead to the specified column.
func (a *LeadMoveToolAdapter) MoveToColumn(ctx context.Context, tenantID, leadID, columnID string) error {
	if err := a.moveLeadUC.Execute(ctx, funnelApp.MoveLeadInput{
		TenantID: tenantID,
		LeadID:   leadID,
		ColumnID: columnID,
	}); err != nil {
		return fmt.Errorf("lead_move_tool_adapter: move lead: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 7. NoteCreatorToolAdapter — satisfies tools.NoteCreator
// ---------------------------------------------------------------------------

// NoteCreatorToolAdapter wraps LeadNoteRepository and LeadRepository.
type NoteCreatorToolAdapter struct {
	leadRepo funnelDomain.LeadRepository
	noteRepo funnelDomain.LeadNoteRepository
}

// NewNoteCreatorToolAdapter creates a NoteCreatorToolAdapter.
func NewNoteCreatorToolAdapter(leadRepo funnelDomain.LeadRepository, noteRepo funnelDomain.LeadNoteRepository) *NoteCreatorToolAdapter {
	return &NoteCreatorToolAdapter{leadRepo: leadRepo, noteRepo: noteRepo}
}

// CreateNote verifies tenant ownership and creates the note.
func (a *NoteCreatorToolAdapter) CreateNote(ctx context.Context, tenantID, leadID, content, createdBy string) error {
	// Verify the lead belongs to this tenant.
	lead, err := a.leadRepo.FindByID(ctx, leadID)
	if err != nil {
		if errors.Is(err, funnelDomain.ErrLeadNotFound) {
			return funnelDomain.ErrLeadNotFound
		}
		return fmt.Errorf("note_creator_tool_adapter: find lead: %w", err)
	}
	if lead.TenantID != tenantID {
		return funnelDomain.ErrLeadNotFound
	}

	note, err := funnelDomain.NewLeadNote(uuid.New().String(), leadID, tenantID, content, createdBy)
	if err != nil {
		return fmt.Errorf("note_creator_tool_adapter: create note: %w", err)
	}
	if err := a.noteRepo.Create(ctx, note); err != nil {
		return fmt.Errorf("note_creator_tool_adapter: save note: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 8. ScoreUpdaterToolAdapter — satisfies tools.ScoreUpdater
// ---------------------------------------------------------------------------

// ScoreUpdaterToolAdapter wraps LeadRepository to update a lead's score.
type ScoreUpdaterToolAdapter struct {
	leadRepo funnelDomain.LeadRepository
}

// NewScoreUpdaterToolAdapter creates a ScoreUpdaterToolAdapter.
func NewScoreUpdaterToolAdapter(leadRepo funnelDomain.LeadRepository) *ScoreUpdaterToolAdapter {
	return &ScoreUpdaterToolAdapter{leadRepo: leadRepo}
}

// UpdateScore looks up the lead, verifies tenant ownership, updates score, and persists.
func (a *ScoreUpdaterToolAdapter) UpdateScore(ctx context.Context, tenantID, leadID string, score int) error {
	lead, err := a.leadRepo.FindByID(ctx, leadID)
	if err != nil {
		return fmt.Errorf("score_updater_tool_adapter: find lead: %w", err)
	}
	if lead.TenantID != tenantID {
		return funnelDomain.ErrLeadNotFound
	}
	lead.UpdateScore(score)
	if err := a.leadRepo.Update(ctx, lead); err != nil {
		return fmt.Errorf("score_updater_tool_adapter: update lead: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 9. AutomationTriggerToolAdapter — satisfies tools.AutomationTrigger
// ---------------------------------------------------------------------------

// AutomationTriggerToolAdapter wraps AutomationEngine to trigger by ID.
type AutomationTriggerToolAdapter struct {
	engine *automationApp.AutomationEngine
}

// NewAutomationTriggerToolAdapter creates an AutomationTriggerToolAdapter.
func NewAutomationTriggerToolAdapter(engine *automationApp.AutomationEngine) *AutomationTriggerToolAdapter {
	return &AutomationTriggerToolAdapter{engine: engine}
}

// TriggerManually delegates to AutomationEngine.TriggerByID.
func (a *AutomationTriggerToolAdapter) TriggerManually(ctx context.Context, tenantID, automationID, leadID string) (string, error) {
	return a.engine.TriggerByID(ctx, tenantID, automationID, leadID)
}

// ---------------------------------------------------------------------------
// 10. SpecialistSwitcherToolAdapter — satisfies tools.SpecialistSwitcher
// ---------------------------------------------------------------------------

// SpecialistSwitcherToolAdapter wraps ConversationStateRepository to switch specialist.
type SpecialistSwitcherToolAdapter struct {
	stateRepo aiDomain.ConversationStateRepository
}

// NewSpecialistSwitcherToolAdapter creates a SpecialistSwitcherToolAdapter.
func NewSpecialistSwitcherToolAdapter(stateRepo aiDomain.ConversationStateRepository) *SpecialistSwitcherToolAdapter {
	return &SpecialistSwitcherToolAdapter{stateRepo: stateRepo}
}

// SwitchSpecialist finds the conversation state and updates the specialist ID.
func (a *SpecialistSwitcherToolAdapter) SwitchSpecialist(ctx context.Context, tenantID, conversationID, specialistID string) error {
	state, err := a.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("specialist_switcher_tool_adapter: find state: %w", err)
	}
	state.SpecialistID = specialistID
	if err := a.stateRepo.Update(ctx, state); err != nil {
		return fmt.Errorf("specialist_switcher_tool_adapter: update state: %w", err)
	}
	return nil
}
