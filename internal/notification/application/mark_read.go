package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	"github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
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
//
// Increments crm_notifications_read_total{type=single} once per successful
// call — the concrete NotificationType is not emitted to avoid an extra DB
// read on this hot UI path.
func (uc *MarkReadUseCase) MarkRead(ctx context.Context, id string) error {
	ctx, span := observability.StartSpan(ctx, "notification.usecase.mark_read",
		attribute.String("notification.id", id),
	)
	defer span.End()

	if err := uc.repo.MarkRead(ctx, id); err != nil {
		return err
	}
	infrastructure.NotificationReadTotal.WithLabelValues("single").Inc()
	return nil
}

// MarkAllRead marks all notifications for the given user as read.
//
// Increments crm_notifications_read_total{type=all} once per call, not once
// per underlying notification. Keeps the counter cheap and comparable with
// single-read volume on dashboards.
func (uc *MarkReadUseCase) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	ctx, span := observability.StartSpan(ctx, "notification.usecase.mark_all_read",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	if err := uc.repo.MarkAllRead(ctx, tenantID, userID); err != nil {
		return err
	}
	infrastructure.NotificationReadTotal.WithLabelValues("all").Inc()
	return nil
}
