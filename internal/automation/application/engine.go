package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// AsyncTypes lists the automation types that must be executed asynchronously.
var AsyncTypes = map[domain.AutomationType]bool{
	domain.TypeAutoMessage: true,
}

// AutomationEngine dispatches automation rules when lead events occur.
type AutomationEngine struct {
	autoRepo  domain.AutomationRepository
	logRepo   domain.ExecutionLogRepository
	executors map[domain.AutomationType]domain.Executor
}

// NewAutomationEngine creates a new AutomationEngine with empty executor registry.
func NewAutomationEngine(autoRepo domain.AutomationRepository, logRepo domain.ExecutionLogRepository) *AutomationEngine {
	return &AutomationEngine{
		autoRepo:  autoRepo,
		logRepo:   logRepo,
		executors: make(map[domain.AutomationType]domain.Executor),
	}
}

// RegisterExecutor registers an executor for the given automation type.
func (e *AutomationEngine) RegisterExecutor(t domain.AutomationType, exec domain.Executor) {
	e.executors[t] = exec
}

// OnLeadEvent finds and dispatches all active automations for the given column,
// running async types in goroutines and sync types inline.
func (e *AutomationEngine) OnLeadEvent(ctx context.Context, tenantID, leadID, columnID string) error {
	automations, err := e.autoRepo.FindByTenantAndColumn(ctx, tenantID, columnID)
	if err != nil {
		return err
	}
	for _, auto := range automations {
		exec, ok := e.executors[auto.Type]
		if !ok {
			continue
		}
		autoCopy := auto // avoid loop-variable capture
		if AsyncTypes[auto.Type] {
			go e.executeAndLog(context.Background(), exec, &autoCopy, leadID, tenantID)
		} else {
			e.executeAndLog(ctx, exec, &autoCopy, leadID, tenantID)
		}
	}
	return nil
}

// executeAndLog runs the executor and persists the execution result in the log.
func (e *AutomationEngine) executeAndLog(ctx context.Context, exec domain.Executor, auto *domain.Automation, leadID, tenantID string) {
	err := exec.Execute(ctx, auto, leadID, tenantID)
	status := domain.StatusSuccess
	errMsg := ""
	if err != nil {
		status = domain.StatusError
		errMsg = err.Error()
	}
	log := domain.NewExecutionLog(uuid.New().String(), auto.ID, leadID, tenantID, status, errMsg)
	_ = e.logRepo.Create(ctx, log)
}
