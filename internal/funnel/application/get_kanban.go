package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type GetKanbanInput struct {
	TenantID  string
	FunnelID  string // optional: if empty, use default funnel
	Search    string
	ProductID string // optional: filter leads by product
}

type KanbanColumn struct {
	ID         string
	Name       string
	OrderIndex int
	Type       string
	Color      string
	Leads      []KanbanLead
	LeadCount  int
}

type KanbanLead struct {
	ID             string
	ContactName    string
	ContactPhone   string
	Score          int
	Status         string
	ConversationID string
	ProductID      string
	ProductName    string
}

type KanbanOutput struct {
	FunnelID   string
	FunnelName string
	Columns    []KanbanColumn
	TotalLeads int
}

type GetKanbanUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
	leadRepo   domain.LeadRepository
}

func NewGetKanbanUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository, leadRepo domain.LeadRepository) *GetKanbanUseCase {
	return &GetKanbanUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo, leadRepo: leadRepo}
}

func (uc *GetKanbanUseCase) Execute(ctx context.Context, input GetKanbanInput) (*KanbanOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.get_kanban",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("funnel.id", input.FunnelID),
	)
	defer span.End()

	var funnel *domain.Funnel
	var err error

	if input.FunnelID != "" {
		funnel, err = uc.funnelRepo.FindByID(ctx, input.FunnelID)
		if err != nil {
			return nil, err
		}
		if funnel.TenantID != input.TenantID {
			return nil, domain.ErrFunnelNotFound
		}
	} else {
		funnel, err = uc.funnelRepo.FindDefaultByTenantID(ctx, input.TenantID)
		if err != nil {
			return nil, err
		}
	}

	columns, err := uc.columnRepo.FindByFunnelID(ctx, funnel.ID)
	if err != nil {
		return nil, err
	}

	kanbanCols := make([]KanbanColumn, len(columns))
	totalLeads := 0

	for i, col := range columns {
		leadList, err := uc.leadRepo.FindByFunnelID(ctx, funnel.ID, domain.LeadFilter{
			Search:    input.Search,
			ColumnID:  col.ID,
			ProductID: input.ProductID,
			Page:      1,
			Limit:     100,
		})
		if err != nil {
			return nil, err
		}

		leads := make([]KanbanLead, len(leadList.Leads))
		for j, lc := range leadList.Leads {
			leads[j] = KanbanLead{
				ID:             lc.Lead.ID,
				ContactName:    lc.ContactName,
				ContactPhone:   lc.ContactPhone,
				Score:          lc.Lead.Score,
				Status:         string(lc.Lead.Status),
				ConversationID: lc.Lead.ConversationID,
				ProductID:      lc.Lead.ProductID,
			}
		}

		kanbanCols[i] = KanbanColumn{
			ID:         col.ID,
			Name:       col.Name,
			OrderIndex: col.OrderIndex,
			Type:       string(col.Type),
			Color:      col.Color,
			Leads:      leads,
			LeadCount:  int(leadList.Total),
		}
		totalLeads += int(leadList.Total)
	}

	return &KanbanOutput{
		FunnelID:   funnel.ID,
		FunnelName: funnel.Name,
		Columns:    kanbanCols,
		TotalLeads: totalLeads,
	}, nil
}
