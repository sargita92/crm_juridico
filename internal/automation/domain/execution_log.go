package domain

import "time"

// ExecutionStatus represents the outcome of an automation execution.
type ExecutionStatus string

const (
	StatusSuccess ExecutionStatus = "success"
	StatusError   ExecutionStatus = "error"
)

// ExecutionLog records the result of a single automation execution for a lead.
type ExecutionLog struct {
	ID           string
	AutomationID string
	LeadID       string
	TenantID     string
	Status       ExecutionStatus
	ErrorMessage string
	ExecutedAt   time.Time
}

// NewExecutionLog constructs an ExecutionLog entity.
func NewExecutionLog(id, automationID, leadID, tenantID string, status ExecutionStatus, errorMessage string) *ExecutionLog {
	return &ExecutionLog{
		ID:           id,
		AutomationID: automationID,
		LeadID:       leadID,
		TenantID:     tenantID,
		Status:       status,
		ErrorMessage: errorMessage,
		ExecutedAt:   time.Now(),
	}
}
