package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// ViewProfileInput is the data required to set a view profile for a group.
type ViewProfileInput struct {
	GroupID        string
	FunnelID       string
	VisibleColumns []string
}

// ViewProfileOutput is the read model returned by view profile queries.
type ViewProfileOutput struct {
	ID             string
	GroupID        string
	FunnelID       string
	VisibleColumns []string
}

// ManageViewProfilesUseCase manages which kanban columns are visible per group
// per funnel.
type ManageViewProfilesUseCase struct {
	profiles domain.ViewProfileRepository
}

func NewManageViewProfilesUseCase(profiles domain.ViewProfileRepository) *ManageViewProfilesUseCase {
	return &ManageViewProfilesUseCase{profiles: profiles}
}

// SetViewProfile creates or updates the view profile for the given group+funnel
// combination.
func (uc *ManageViewProfilesUseCase) SetViewProfile(ctx context.Context, input ViewProfileInput) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.set_view_profile",
		attribute.String("group.id", input.GroupID),
		attribute.String("funnel.id", input.FunnelID),
	)
	defer span.End()

	existing, err := uc.profiles.FindByGroupAndFunnel(ctx, input.GroupID, input.FunnelID)
	if err != nil && !errors.Is(err, domain.ErrViewProfileNotFound) {
		return err
	}

	if existing != nil {
		existing.UpdateColumns(input.VisibleColumns)
		if err := uc.profiles.CreateOrUpdate(ctx, existing); err != nil {
			return err
		}
		infrastructure.ChangesTotal.WithLabelValues("view_profile", "updated").Inc()
		return nil
	}

	vp, err := domain.NewViewProfile(uuid.New().String(), input.GroupID, input.FunnelID, input.VisibleColumns)
	if err != nil {
		return err
	}
	if err := uc.profiles.CreateOrUpdate(ctx, vp); err != nil {
		return err
	}
	infrastructure.ChangesTotal.WithLabelValues("view_profile", "updated").Inc()
	return nil
}

// ListByGroup returns all view profiles associated with the given group.
func (uc *ManageViewProfilesUseCase) ListByGroup(ctx context.Context, groupID string) ([]ViewProfileOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.list_view_profiles",
		attribute.String("group.id", groupID),
	)
	defer span.End()

	list, err := uc.profiles.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]ViewProfileOutput, len(list))
	for i, vp := range list {
		out[i] = ViewProfileOutput{
			ID:             vp.ID,
			GroupID:        vp.GroupID,
			FunnelID:       vp.FunnelID,
			VisibleColumns: vp.VisibleColumns,
		}
	}
	return out, nil
}

// ResolveVisibleColumns returns the union of visible columns for the given
// groups and funnel. A nil return value means "show all" (no restrictions
// found). An empty non-nil slice means no columns are visible.
func (uc *ManageViewProfilesUseCase) ResolveVisibleColumns(ctx context.Context, groupIDs []string, funnelID string) ([]string, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.resolve_visible_columns",
		attribute.String("funnel.id", funnelID),
		attribute.Int("groups.count", len(groupIDs)),
	)
	defer span.End()

	if len(groupIDs) == 0 {
		return nil, nil
	}

	relevant, err := uc.profiles.FindByGroupIDs(ctx, groupIDs, funnelID)
	if err != nil {
		return nil, err
	}

	// No profiles configured: no column restrictions (show all).
	if len(relevant) == 0 {
		return nil, nil
	}

	// Build the union of visible columns across all matching profiles.
	// Use a non-nil empty slice to distinguish "restricted to zero columns"
	// from "no restriction" (nil).
	seen := make(map[string]bool)
	union := make([]string, 0)
	for _, vp := range relevant {
		for _, col := range vp.VisibleColumns {
			if !seen[col] {
				seen[col] = true
				union = append(union, col)
			}
		}
	}
	return union, nil
}
