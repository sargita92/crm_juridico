package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// MarkReadUseCase handles marking notifications as read.
type MarkReadUseCase struct {
	repo domain.NotificationRepository
}

func NewMarkReadUseCase(repo domain.NotificationRepository) *MarkReadUseCase {
	return &MarkReadUseCase{repo: repo}
}

// MarkRead marks a single notification as read by its ID.
func (uc *MarkReadUseCase) MarkRead(ctx context.Context, id string) error {
	ctx, span := observability.StartSpan(ctx, "notification.usecase.mark_read",
		attribute.String("notification.id", id),
	)
	defer span.End()

	return uc.repo.MarkRead(ctx, id)
}

// MarkAllRead marks all notifications for the given user as read.
func (uc *MarkReadUseCase) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	ctx, span := observability.StartSpan(ctx, "notification.usecase.mark_all_read",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	return uc.repo.MarkAllRead(ctx, tenantID, userID)
}
