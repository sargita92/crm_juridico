package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// NotificationOutput is the read model returned to callers.
type NotificationOutput struct {
	ID        string
	TenantID  string
	UserID    string
	Type      domain.NotificationType
	Title     string
	Body      string
	Metadata  map[string]string
	Read      bool
	CreatedAt time.Time
}

// ListNotificationsUseCase retrieves notifications for a user.
type ListNotificationsUseCase struct {
	repo domain.NotificationRepository
}

func NewListNotificationsUseCase(repo domain.NotificationRepository) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{repo: repo}
}

// Execute lists notifications for the given user with optional unread filter and pagination.
func (uc *ListNotificationsUseCase) Execute(ctx context.Context, tenantID, userID string, onlyUnread bool, limit, offset int) ([]NotificationOutput, error) {
	items, err := uc.repo.FindByUserID(ctx, tenantID, userID, onlyUnread, limit, offset)
	if err != nil {
		return nil, err
	}
	outputs := make([]NotificationOutput, len(items))
	for i, n := range items {
		outputs[i] = toNotificationOutput(n)
	}
	return outputs, nil
}

// CountUnread returns the number of unread notifications for the given user.
func (uc *ListNotificationsUseCase) CountUnread(ctx context.Context, tenantID, userID string) (int64, error) {
	return uc.repo.CountUnread(ctx, tenantID, userID)
}

func toNotificationOutput(n domain.Notification) NotificationOutput {
	return NotificationOutput{
		ID:        n.ID,
		TenantID:  n.TenantID,
		UserID:    n.UserID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Metadata:  n.Metadata,
		Read:      n.Read,
		CreatedAt: n.CreatedAt,
	}
}
