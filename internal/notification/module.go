package notification

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	notificationdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"
	"github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
	notifhttp "github.com/sasrgita/crm-juridico/internal/notification/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Module wires up the notification domain.
type Module struct {
	handler       *notifhttp.Handler
	pageHandler   *notifhttp.PageHandler
	notifyService *application.NotifyService
}

// NewModule builds and returns a fully wired notification Module.
//
// If the supplied EventBus also satisfies events.GlobalEventBus, a background
// goroutine subscribes to cross-tenant events and creates persistent
// notifications in response to EventLeadResponsibleAssigned.
func NewModule(db *gorm.DB, eventBus events.EventBus, log *zap.Logger) *Module {
	notifRepo := infrastructure.NewGormNotificationRepository(db)
	prefRepo := infrastructure.NewGormPreferenceRepository(db)

	notifyService := application.NewNotifyService(notifRepo, prefRepo, eventBus)
	listUC := application.NewListNotificationsUseCase(notifRepo)
	markReadUC := application.NewMarkReadUseCase(notifRepo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	// Renderer is nil here because templates haven't been parsed yet.
	// main.go calls SetRenderer(...) after setupRouter() completes.
	handler := notifhttp.NewHandler(notifyService, listUC, markReadUC, prefsUC, eventBus, nil, log)
	pageHandler := notifhttp.NewPageHandler(listUC, markReadUC, log)

	// Subscribe to cross-module events that must produce user notifications.
	// Requires a GlobalEventBus (cross-tenant). Busses that only support the
	// per-tenant Subscribe method are skipped with an Info log.
	//
	// The subscription is registered SYNCHRONOUSLY before NewModule returns so
	// that any event published immediately after construction is guaranteed to
	// be delivered — no race between NewModule and goroutine scheduling.
	if globalBus, ok := eventBus.(events.GlobalEventBus); ok {
		ch, unsub := globalBus.SubscribeAll()
		go consumeResponsibleAssigned(ch, unsub, notifyService, log)
	} else {
		log.Info("notification: event bus does not support cross-tenant subscription; skipping lead-assigned listener")
	}

	return &Module{handler: handler, pageHandler: pageHandler, notifyService: notifyService}
}

// consumeResponsibleAssigned drains the global subscription channel and turns
// EventLeadResponsibleAssigned events into persistent notifications for the
// responsible user. It exits cleanly when the channel closes.
func consumeResponsibleAssigned(ch <-chan events.Event, unsub func(), svc *application.NotifyService, log *zap.Logger) {
	defer unsub()

	for ev := range ch {
		if ev.Type != events.EventLeadResponsibleAssigned {
			continue
		}
		payload, ok := ev.Payload.(events.ResponsibleAssignedPayload)
		if !ok {
			log.Warn("notification: unexpected payload shape for responsible_assigned",
				zap.Any("payload", ev.Payload),
			)
			continue
		}

		metadata := map[string]string{
			"lead_id":   payload.LeadID,
			"reason":    payload.Reason,
			"outcome":   payload.Outcome,
			"algorithm": payload.Algorithm,
		}

		// Listener runs off the request path; use Background context.
		err := svc.Notify(
			context.Background(),
			payload.ResponsibleUserID,
			ev.TenantID,
			notificationdomain.TypeLeadAssigned,
			"Novo lead atribuído",
			"Você recebeu um novo lead.",
			metadata,
		)
		if err != nil {
			log.Error("notification: failed to create lead_assigned notification",
				zap.String("tenant_id", ev.TenantID),
				zap.String("lead_id", payload.LeadID),
				zap.Error(err),
			)
		}
	}
}

// Name implements module.Module.
func (m *Module) Name() string { return "notification" }

// RegisterRoutes implements module.Module.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
	m.pageHandler.RegisterPageRoutes(router, mw.Auth, mw.Tenant)
}

// NotifyService exposes the NotifyService for cross-module use.
func (m *Module) NotifyService() *application.NotifyService {
	return m.notifyService
}

// SetRenderer injects the ToastRenderer after templates are parsed.
// Called by main.go after setupRouter() completes.
func (m *Module) SetRenderer(r *notifhttp.ToastRenderer) {
	m.handler.SetRenderer(r)
}
