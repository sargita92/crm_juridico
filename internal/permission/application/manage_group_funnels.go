package application

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// GroupFunnelInput is the data required to associate a group with a funnel.
type GroupFunnelInput struct {
	GroupID   string
	FunnelID  string
	ColumnIDs []string
}

// GroupFunnelOutput is the read model returned by group funnel queries.
type GroupFunnelOutput struct {
	ID        string
	GroupID   string
	FunnelID  string
	ColumnIDs []string
}

// ManageGroupFunnelsUseCase handles assigning funnel access to permission groups.
type ManageGroupFunnelsUseCase struct {
	funnels domain.GroupFunnelRepository
}

func NewManageGroupFunnelsUseCase(funnels domain.GroupFunnelRepository) *ManageGroupFunnelsUseCase {
	return &ManageGroupFunnelsUseCase{funnels: funnels}
}

// SetGroupFunnel creates or updates the funnel association for the given group.
func (uc *ManageGroupFunnelsUseCase) SetGroupFunnel(ctx context.Context, input GroupFunnelInput) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.set_group_funnel",
		attribute.String("group.id", input.GroupID),
		attribute.String("funnel.id", input.FunnelID),
	)
	defer span.End()

	gf, err := domain.NewGroupFunnel(uuid.New().String(), input.GroupID, input.FunnelID, input.ColumnIDs)
	if err != nil {
		return err
	}
	if err := uc.funnels.CreateOrUpdate(ctx, gf); err != nil {
		return err
	}
	infrastructure.ChangesTotal.WithLabelValues("funnel", "updated").Inc()
	return nil
}

// ListByGroup returns all funnel associations for the given group.
func (uc *ManageGroupFunnelsUseCase) ListByGroup(ctx context.Context, groupID string) ([]GroupFunnelOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.list_group_funnels",
		attribute.String("group.id", groupID),
	)
	defer span.End()

	list, err := uc.funnels.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]GroupFunnelOutput, len(list))
	for i, gf := range list {
		out[i] = GroupFunnelOutput{
			ID:        gf.ID,
			GroupID:   gf.GroupID,
			FunnelID:  gf.FunnelID,
			ColumnIDs: gf.ColumnIDs,
		}
	}
	return out, nil
}

// RemoveGroupFunnel deletes a group-funnel association by groupID and funnelID.
func (uc *ManageGroupFunnelsUseCase) RemoveGroupFunnel(ctx context.Context, groupID, funnelID string) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.remove_group_funnel",
		attribute.String("group.id", groupID),
		attribute.String("funnel.id", funnelID),
	)
	defer span.End()

	return uc.funnels.Delete(ctx, groupID, funnelID)
}
