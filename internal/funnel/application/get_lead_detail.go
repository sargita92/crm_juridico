package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GetLeadDetailInput struct {
	TenantID string
	LeadID   string
}

type MessageSummaryOutput struct {
	Direction string
	Content   string
	Timestamp time.Time
}

type LeadNoteOutput struct {
	ID            string
	Content       string
	CreatedBy     string
	CreatedByName string
	CreatedAt     time.Time
}

type LeadMovementOutput struct {
	ID             string
	FromColumnID   string
	FromColumnName string
	ToColumnID     string
	ToColumnName   string
	MovedAt        time.Time
}

type LeadDetailOutput struct {
	ID              string
	TenantID        string
	FunnelID        string
	FunnelName      string
	ColumnID        string
	ColumnName      string
	ContactID       string
	ContactName     string
	ContactPhone    string
	ConversationID  string
	Score           int
	Status          string
	ColumnEnteredAt time.Time
	CreatedAt       time.Time
	Messages        []MessageSummaryOutput
	Movements       []LeadMovementOutput
	Notes           []LeadNoteOutput
	ProductName     string
	AssignedToName  string
}

type GetLeadDetailUseCase struct {
	leadRepo         domain.LeadRepository
	movementRepo     domain.LeadMovementRepository
	funnelRepo       domain.FunnelRepository
	columnRepo       domain.ColumnRepository
	contactProvider  domain.ContactProvider
	messageProvider  domain.MessageProvider
	noteRepo         domain.LeadNoteRepository
	userNameProvider domain.UserNameProvider
	productProvider  domain.ProductProvider
}

func NewGetLeadDetailUseCase(
	leadRepo domain.LeadRepository,
	movementRepo domain.LeadMovementRepository,
	funnelRepo domain.FunnelRepository,
	columnRepo domain.ColumnRepository,
	contactProvider domain.ContactProvider,
	messageProvider domain.MessageProvider,
	noteRepo domain.LeadNoteRepository,
	userNameProvider domain.UserNameProvider,
	productProvider domain.ProductProvider,
) *GetLeadDetailUseCase {
	return &GetLeadDetailUseCase{
		leadRepo:         leadRepo,
		movementRepo:     movementRepo,
		funnelRepo:       funnelRepo,
		columnRepo:       columnRepo,
		contactProvider:  contactProvider,
		messageProvider:  messageProvider,
		noteRepo:         noteRepo,
		userNameProvider: userNameProvider,
		productProvider:  productProvider,
	}
}

func (uc *GetLeadDetailUseCase) Execute(ctx context.Context, input GetLeadDetailInput) (*LeadDetailOutput, error) {
	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead.TenantID != input.TenantID {
		return nil, domain.ErrLeadNotFound
	}

	// Funnel name
	var funnelName string
	if funnel, err := uc.funnelRepo.FindByID(ctx, lead.FunnelID); err == nil {
		funnelName = funnel.Name
	}

	// Column name
	var columnName string
	if col, err := uc.columnRepo.FindByID(ctx, lead.ColumnID); err == nil {
		columnName = col.Name
	}

	// Contact info
	var contactName, contactPhone string
	if info, err := uc.contactProvider.FindByID(ctx, lead.ContactID); err == nil {
		contactName = info.Name
		contactPhone = info.Phone
	}

	// Messages
	var messages []MessageSummaryOutput
	if msgs, err := uc.messageProvider.FindRecentByConversationID(ctx, lead.ConversationID, 10); err == nil {
		messages = make([]MessageSummaryOutput, len(msgs))
		for i, m := range msgs {
			messages[i] = MessageSummaryOutput{
				Direction: m.Direction,
				Content:   m.Content,
				Timestamp: m.Timestamp,
			}
		}
	}

	// Movements with column names
	movements, err := uc.movementRepo.FindByLeadID(ctx, lead.ID)
	if err != nil {
		return nil, err
	}
	mvOutputs := make([]LeadMovementOutput, len(movements))
	for i, mv := range movements {
		var fromName, toName string
		if mv.FromColumnID != "" {
			if col, err := uc.columnRepo.FindByID(ctx, mv.FromColumnID); err == nil {
				fromName = col.Name
			}
		}
		if col, err := uc.columnRepo.FindByID(ctx, mv.ToColumnID); err == nil {
			toName = col.Name
		}
		mvOutputs[i] = LeadMovementOutput{
			ID:             mv.ID,
			FromColumnID:   mv.FromColumnID,
			FromColumnName: fromName,
			ToColumnID:     mv.ToColumnID,
			ToColumnName:   toName,
			MovedAt:        mv.MovedAt,
		}
	}

	// Notes
	var noteOutputs []LeadNoteOutput
	if notes, err := uc.noteRepo.FindByLeadID(ctx, lead.ID); err == nil {
		noteOutputs = make([]LeadNoteOutput, len(notes))
		for i, n := range notes {
			var createdByName string
			if name, err := uc.userNameProvider.FindNameByID(ctx, n.CreatedBy); err == nil {
				createdByName = name
			}
			noteOutputs[i] = LeadNoteOutput{
				ID:            n.ID,
				Content:       n.Content,
				CreatedBy:     n.CreatedBy,
				CreatedByName: createdByName,
				CreatedAt:     n.CreatedAt,
			}
		}
	}

	// Product name
	var productName string
	if lead.ProductID != "" && uc.productProvider != nil {
		if name, err := uc.productProvider.FindProductNameByID(ctx, lead.ProductID); err == nil {
			productName = name
		}
	}

	return &LeadDetailOutput{
		ID:              lead.ID,
		TenantID:        lead.TenantID,
		FunnelID:        lead.FunnelID,
		FunnelName:      funnelName,
		ColumnID:        lead.ColumnID,
		ColumnName:      columnName,
		ContactID:       lead.ContactID,
		ContactName:     contactName,
		ContactPhone:    contactPhone,
		ConversationID:  lead.ConversationID,
		Score:           lead.Score,
		Status:          string(lead.Status),
		ColumnEnteredAt: lead.ColumnEnteredAt,
		CreatedAt:       lead.CreatedAt,
		Messages:        messages,
		Movements:       mvOutputs,
		Notes:           noteOutputs,
		ProductName:     productName,
	}, nil
}
