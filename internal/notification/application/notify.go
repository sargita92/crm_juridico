package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	notifinfra "github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
	events "github.com/sasrgita/crm-juridico/internal/shared/events"
)

// NotifyService creates a notification and publishes an in-app event.
type NotifyService struct {
	notifRepo domain.NotificationRepository
	prefRepo  domain.PreferenceRepository
	eventBus  events.EventBus
}

func NewNotifyService(
	notifRepo domain.NotificationRepository,
	prefRepo domain.PreferenceRepository,
	eventBus events.EventBus,
) *NotifyService {
	return &NotifyService{
		notifRepo: notifRepo,
		prefRepo:  prefRepo,
		eventBus:  eventBus,
	}
}

// Notify creates a persistent notification and publishes it to the event bus.
func (s *NotifyService) Notify(
	ctx context.Context,
	userID, tenantID string,
	notifType domain.NotificationType,
	title, body string,
	metadata map[string]string,
) error {
	notif, err := domain.NewNotification(uuid.New().String(), tenantID, userID, notifType, title, body, metadata)
	if err != nil {
		return err
	}
	if err := s.notifRepo.Create(ctx, notif); err != nil {
		return err
	}
	notifinfra.NotificationsDeliveredTotal.WithLabelValues(string(notif.Type)).Inc()
	// Always publish in-app event for SSE consumers.
	s.eventBus.Publish(events.Event{
		Type:     events.EventNotification,
		TenantID: tenantID,
		Payload:  notif,
	})
	return nil
}
