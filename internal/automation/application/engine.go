package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
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
	ctx, span := observability.StartSpan(ctx, "automation.engine.on_lead_event",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("funnel.column.id", columnID),
	)
	defer span.End()

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

// TriggerByID executes a single automation by its ID for the given lead.
// It is intended for manual/AI-initiated triggers (e.g., via the trigger_automation tool).
// Returns a human-readable summary or an error if the automation or its executor is missing.
func (e *AutomationEngine) TriggerByID(ctx context.Context, tenantID, automationID, leadID string) (string, error) {
	ctx, span := observability.StartSpan(ctx, "automation.engine.trigger_by_id",
		attribute.String("tenant.id", tenantID),
		attribute.String("automation.id", automationID),
		attribute.String("lead.id", leadID),
	)
	defer span.End()

	auto, err := e.autoRepo.FindByID(ctx, automationID)
	if err != nil {
		return "", fmt.Errorf("automation_engine: find automation %s: %w", automationID, err)
	}
	if auto.TenantID != tenantID {
		return "", fmt.Errorf("automation_engine: automation %s not found for tenant", automationID)
	}
	exec, ok := e.executors[auto.Type]
	if !ok {
		return "", fmt.Errorf("automation_engine: no executor registered for type %s", auto.Type)
	}
	e.executeAndLog(ctx, exec, auto, leadID, tenantID)
	return fmt.Sprintf("automacao %s executada com sucesso para lead %s", automationID, leadID), nil
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
	infrastructure.ExecutionsTotal.WithLabelValues(string(auto.Type), string(status)).Inc()
	log := domain.NewExecutionLog(uuid.New().String(), auto.ID, leadID, tenantID, status, errMsg)
	_ = e.logRepo.Create(ctx, log)
}
