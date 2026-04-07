package application

import (
	"context"
	"sync"
	"time"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// --- AutomationRepository mock ---

type mockAutoRepo struct {
	mu          sync.Mutex
	automations []domain.Automation
}

func (m *mockAutoRepo) Create(_ context.Context, a *domain.Automation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.automations = append(m.automations, *a)
	return nil
}

func (m *mockAutoRepo) FindByID(_ context.Context, id string) (*domain.Automation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range m.automations {
		if a.ID == id {
			cp := a
			return &cp, nil
		}
	}
	return nil, domain.ErrAutomationNotFound
}

func (m *mockAutoRepo) Update(_ context.Context, a *domain.Automation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.automations {
		if existing.ID == a.ID {
			m.automations[i] = *a
			return nil
		}
	}
	return domain.ErrAutomationNotFound
}

func (m *mockAutoRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, a := range m.automations {
		if a.ID == id {
			m.automations = append(m.automations[:i], m.automations[i+1:]...)
			return nil
		}
	}
	return domain.ErrAutomationNotFound
}

func (m *mockAutoRepo) FindByTenantAndColumn(_ context.Context, tenantID, columnID string) ([]domain.Automation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Automation
	for _, a := range m.automations {
		if a.TenantID == tenantID && a.ColumnID == columnID && a.Active {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAutoRepo) FindByFunnelID(_ context.Context, tenantID, funnelID string) ([]domain.Automation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Automation
	for _, a := range m.automations {
		if a.TenantID == tenantID && a.FunnelID == funnelID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAutoRepo) FindActiveByType(_ context.Context, t domain.AutomationType) ([]domain.Automation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Automation
	for _, a := range m.automations {
		if a.Type == t && a.Active {
			result = append(result, a)
		}
	}
	return result, nil
}

// --- ExecutionLogRepository mock ---

type mockLogRepo struct {
	mu   sync.Mutex
	logs []*domain.ExecutionLog
}

func (m *mockLogRepo) Create(_ context.Context, log *domain.ExecutionLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockLogRepo) FindByAutomationID(_ context.Context, automationID string, limit, offset int) ([]domain.ExecutionLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.ExecutionLog
	for _, l := range m.logs {
		if l.AutomationID == automationID {
			result = append(result, *l)
		}
	}
	return result, nil
}

func (m *mockLogRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs)
}

func (m *mockLogRepo) last() *domain.ExecutionLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.logs) == 0 {
		return nil
	}
	return m.logs[len(m.logs)-1]
}

// waitForCount blocks until the log reaches at least n entries or the timeout elapses.
func (m *mockLogRepo) waitForCount(n int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if m.count() >= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// --- Executor mock ---

type mockExecutor struct {
	mu        sync.Mutex
	callCount int
	err       error
}

func (m *mockExecutor) Execute(_ context.Context, _ *domain.Automation, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++
	return m.err
}

func (m *mockExecutor) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}
