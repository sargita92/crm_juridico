package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
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
	gf, err := domain.NewGroupFunnel(uuid.New().String(), input.GroupID, input.FunnelID, input.ColumnIDs)
	if err != nil {
		return err
	}
	return uc.funnels.CreateOrUpdate(ctx, gf)
}

// ListByGroup returns all funnel associations for the given group.
func (uc *ManageGroupFunnelsUseCase) ListByGroup(ctx context.Context, groupID string) ([]GroupFunnelOutput, error) {
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
	return uc.funnels.Delete(ctx, groupID, funnelID)
}
