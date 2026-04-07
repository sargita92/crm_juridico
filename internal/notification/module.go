package notification

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
	notifhttp "github.com/sasrgita/crm-juridico/internal/notification/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Module wires up the notification domain.
type Module struct {
	handler       *notifhttp.Handler
	notifyService *application.NotifyService
}

// NewModule builds and returns a fully wired notification Module.
func NewModule(db *gorm.DB, eventBus events.EventBus, log *zap.Logger) *Module {
	notifRepo := infrastructure.NewGormNotificationRepository(db)
	prefRepo := infrastructure.NewGormPreferenceRepository(db)

	notifyService := application.NewNotifyService(notifRepo, prefRepo, eventBus)
	listUC := application.NewListNotificationsUseCase(notifRepo)
	markReadUC := application.NewMarkReadUseCase(notifRepo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	handler := notifhttp.NewHandler(notifyService, listUC, markReadUC, prefsUC, eventBus, log)

	return &Module{handler: handler, notifyService: notifyService}
}

// Name implements module.Module.
func (m *Module) Name() string { return "notification" }

// RegisterRoutes implements module.Module.
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
}

// NotifyService exposes the NotifyService for cross-module use.
func (m *Module) NotifyService() *application.NotifyService {
	return m.notifyService
}
