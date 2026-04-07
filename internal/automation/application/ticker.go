package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/automation/application/executors"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// expirationRunner is the minimal interface ExpirationTicker needs from the expiration executor.
// Using an interface here allows tests to inject a mock without depending on the concrete type.
type expirationRunner interface {
	FindExpiredLeads(ctx context.Context, a *domain.Automation) ([]string, error)
	Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error
}

// ExpirationTicker polls active expiration automations on a fixed interval and
// processes any leads that have exceeded their configured duration.
type ExpirationTicker struct {
	autoRepo    domain.AutomationRepository
	logRepo     domain.ExecutionLogRepository
	expExecutor expirationRunner
	interval    time.Duration
	stop        chan struct{}
}

// NewExpirationTicker creates an ExpirationTicker that polls every 5 minutes.
func NewExpirationTicker(
	autoRepo domain.AutomationRepository,
	logRepo domain.ExecutionLogRepository,
	expExecutor *executors.ExpirationExecutor,
) *ExpirationTicker {
	return &ExpirationTicker{
		autoRepo:    autoRepo,
		logRepo:     logRepo,
		expExecutor: expExecutor,
		interval:    5 * time.Minute,
		stop:        make(chan struct{}),
	}
}

// Start launches the background polling goroutine.
func (t *ExpirationTicker) Start() {
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.tick()
			case <-t.stop:
				return
			}
		}
	}()
}

// Stop signals the background goroutine to stop.
func (t *ExpirationTicker) Stop() { close(t.stop) }

// tick processes all active expiration automations for one polling cycle.
func (t *ExpirationTicker) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	automations, err := t.autoRepo.FindActiveByType(ctx, domain.TypeExpiration)
	if err != nil {
		return
	}

	for _, auto := range automations {
		autoCopy := auto // avoid loop-variable capture
		leadIDs, err := t.expExecutor.FindExpiredLeads(ctx, &autoCopy)
		if err != nil {
			continue
		}
		for _, leadID := range leadIDs {
			execErr := t.expExecutor.Execute(ctx, &autoCopy, leadID, autoCopy.TenantID)
			status := domain.StatusSuccess
			errMsg := ""
			if execErr != nil {
				status = domain.StatusError
				errMsg = execErr.Error()
			}
			log := domain.NewExecutionLog(uuid.New().String(), autoCopy.ID, leadID, autoCopy.TenantID, status, errMsg)
			_ = t.logRepo.Create(ctx, log)
		}
	}
}
