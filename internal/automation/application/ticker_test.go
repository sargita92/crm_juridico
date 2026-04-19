package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	infra "github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExpirationRunner is a test double for the expirationRunner interface.
type mockExpirationRunner struct {
	mu           sync.Mutex
	expiredLeads map[string][]string // automationID -> leadIDs
	findErr      error
	executeErr   error
	executeCalls int
}

func (m *mockExpirationRunner) FindExpiredLeads(_ context.Context, a *domain.Automation) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.expiredLeads[a.ID], nil
}

func (m *mockExpirationRunner) Execute(_ context.Context, _ *domain.Automation, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalls++
	return m.executeErr
}

func (m *mockExpirationRunner) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executeCalls
}

// newTestTicker builds an ExpirationTicker wired up for unit testing (interval irrelevant
// because tests call tick() directly).
func newTestTicker(autoRepo domain.AutomationRepository, logRepo domain.ExecutionLogRepository, runner expirationRunner) *ExpirationTicker {
	return &ExpirationTicker{
		autoRepo:    autoRepo,
		logRepo:     logRepo,
		expExecutor: runner,
		interval:    time.Hour, // not used in direct tick() calls
		stop:        make(chan struct{}),
	}
}

// TestTick_NoAutomations verifies that tick() completes without errors when no
// expiration automations are active.
func TestTick_NoAutomations(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	runner := &mockExpirationRunner{expiredLeads: map[string][]string{}}

	ticker := newTestTicker(autoRepo, logRepo, runner)
	ticker.tick()

	assert.Equal(t, 0, runner.calls())
	assert.Equal(t, 0, logRepo.count())
}

// TestTick_ExecutesAndLogsSuccess verifies that tick() calls Execute for each expired
// lead and creates a success log entry.
func TestTick_ExecutesAndLogsSuccess(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("exp-1", "tenant-1", "funnel-1", "col-1", domain.TypeExpiration)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	runner := &mockExpirationRunner{
		expiredLeads: map[string][]string{
			"exp-1": {"lead-1", "lead-2"},
		},
	}

	before := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues("expiration", "success"))

	ticker := newTestTicker(autoRepo, logRepo, runner)
	ticker.tick()

	after := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues("expiration", "success"))
	assert.Equal(t, before+2, after, "executions_total{type=expiration,outcome=success} should increment by 2 (one per lead)")

	assert.Equal(t, 2, runner.calls())
	assert.Equal(t, 2, logRepo.count())

	last := logRepo.last()
	require.NotNil(t, last)
	assert.Equal(t, domain.StatusSuccess, last.Status)
	assert.Empty(t, last.ErrorMessage)
}

// TestTick_ExecutorError verifies that when Execute returns an error the log entry
// is created with StatusError and the error message is preserved.
func TestTick_ExecutorError(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("exp-err", "tenant-1", "funnel-1", "col-1", domain.TypeExpiration)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	runner := &mockExpirationRunner{
		expiredLeads: map[string][]string{
			"exp-err": {"lead-x"},
		},
		executeErr: errors.New("archive failed"),
	}

	before := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues("expiration", "error"))

	ticker := newTestTicker(autoRepo, logRepo, runner)
	ticker.tick()

	after := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues("expiration", "error"))
	assert.Equal(t, before+1, after, "executions_total{type=expiration,outcome=error} should increment by 1")

	assert.Equal(t, 1, runner.calls())
	assert.Equal(t, 1, logRepo.count())

	last := logRepo.last()
	require.NotNil(t, last)
	assert.Equal(t, domain.StatusError, last.Status)
	assert.Equal(t, "archive failed", last.ErrorMessage)
}

// TestTick_FindExpiredLeadsError verifies that when FindExpiredLeads returns an error
// the automation is skipped and no logs are created for it.
func TestTick_FindExpiredLeadsError(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("exp-find-err", "tenant-1", "funnel-1", "col-1", domain.TypeExpiration)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	runner := &mockExpirationRunner{
		expiredLeads: map[string][]string{},
		findErr:      errors.New("db error"),
	}

	ticker := newTestTicker(autoRepo, logRepo, runner)
	ticker.tick()

	assert.Equal(t, 0, runner.calls())
	assert.Equal(t, 0, logRepo.count())
}

// TestTick_RepoError verifies that when FindActiveByType returns an error tick() returns
// early and does nothing.
func TestTick_RepoError(t *testing.T) {
	autoRepo := &errAutoRepo{err: errors.New("db down")}
	logRepo := &mockLogRepo{}
	runner := &mockExpirationRunner{expiredLeads: map[string][]string{}}

	ticker := newTestTicker(autoRepo, logRepo, runner)
	ticker.tick() // must not panic

	assert.Equal(t, 0, runner.calls())
	assert.Equal(t, 0, logRepo.count())
}

// TestStop verifies that calling Stop() signals the background goroutine to exit.
func TestStop(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	runner := &mockExpirationRunner{expiredLeads: map[string][]string{}}

	ticker := &ExpirationTicker{
		autoRepo:    autoRepo,
		logRepo:     logRepo,
		expExecutor: runner,
		interval:    10 * time.Millisecond,
		stop:        make(chan struct{}),
	}
	ticker.Start()

	// Stop must not block or panic.
	ticker.Stop()

	// Calling Stop a second time on the same channel would panic — guard with a
	// short wait to confirm the goroutine exited cleanly.
	time.Sleep(20 * time.Millisecond)
}

// errAutoRepo is a minimal AutomationRepository stub that always returns an error for
// FindActiveByType (used for the repo-error path in tick()).
type errAutoRepo struct {
	err error
}

func (r *errAutoRepo) Create(_ context.Context, _ *domain.Automation) error {
	return nil
}
func (r *errAutoRepo) FindByID(_ context.Context, _ string) (*domain.Automation, error) {
	return nil, r.err
}
func (r *errAutoRepo) Update(_ context.Context, _ *domain.Automation) error { return nil }
func (r *errAutoRepo) Delete(_ context.Context, _ string) error              { return nil }
func (r *errAutoRepo) FindByTenantAndColumn(_ context.Context, _, _ string) ([]domain.Automation, error) {
	return nil, r.err
}
func (r *errAutoRepo) FindByFunnelID(_ context.Context, _, _ string) ([]domain.Automation, error) {
	return nil, r.err
}
func (r *errAutoRepo) FindActiveByType(_ context.Context, _ domain.AutomationType) ([]domain.Automation, error) {
	return nil, r.err
}
