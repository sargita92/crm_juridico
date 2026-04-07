package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
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
	return uc.repo.MarkRead(ctx, id)
}

// MarkAllRead marks all notifications for the given user as read.
func (uc *MarkReadUseCase) MarkAllRead(ctx context.Context, tenantID, userID string) error {
	return uc.repo.MarkAllRead(ctx, tenantID, userID)
}
