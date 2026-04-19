package application

import (
	"context"
	"errors"
	"testing"
	"time"

	infra "github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeAutomation(id, tenantID, funnelID, columnID string, t domain.AutomationType) domain.Automation {
	a, err := domain.NewAutomation(id, tenantID, funnelID, columnID, t, nil, 1)
	if err != nil {
		panic(err)
	}
	return *a
}

// TestEngine_OnLeadEvent_ExecutesAutomations verifies that a registered executor is called
// and the execution is logged with success status.
func TestEngine_OnLeadEvent_ExecutesAutomations(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("auto-1", "tenant-1", "funnel-1", "col-1", domain.TypeAutoNote)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	exec := &mockExecutor{}
	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoNote, exec)

	before := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues(string(a.Type), "success"))

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "col-1")
	require.NoError(t, err)

	after := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues(string(a.Type), "success"))
	assert.Equal(t, before+1, after, "executions_total{type=auto_note,outcome=success} should increment by 1")

	assert.Equal(t, 1, exec.calls(), "executor should have been called once")
	assert.Equal(t, 1, logRepo.count(), "one execution log should exist")

	last := logRepo.last()
	require.NotNil(t, last)
	assert.Equal(t, domain.StatusSuccess, last.Status)
	assert.Equal(t, "lead-1", last.LeadID)
	assert.Equal(t, "tenant-1", last.TenantID)
	assert.Equal(t, "auto-1", last.AutomationID)
}

// TestEngine_OnLeadEvent_NoAutomations verifies that when no automations exist for a column
// nothing is executed and no logs are created.
func TestEngine_OnLeadEvent_NoAutomations(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	exec := &mockExecutor{}
	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoNote, exec)

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "col-nonexistent")
	require.NoError(t, err)

	assert.Equal(t, 0, exec.calls())
	assert.Equal(t, 0, logRepo.count())
}

// TestEngine_OnLeadEvent_ExecutorError verifies that when an executor returns an error
// the log is created with StatusError and the error message is recorded.
func TestEngine_OnLeadEvent_ExecutorError(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("auto-err", "tenant-1", "funnel-1", "col-1", domain.TypeAutoNote)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	exec := &mockExecutor{err: errors.New("boom")}
	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoNote, exec)

	before := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues(string(a.Type), "error"))

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "col-1")
	require.NoError(t, err) // OnLeadEvent itself does not surface executor errors

	after := testutil.ToFloat64(infra.ExecutionsTotal.WithLabelValues(string(a.Type), "error"))
	assert.Equal(t, before+1, after, "executions_total{type=auto_note,outcome=error} should increment by 1")

	assert.Equal(t, 1, exec.calls())
	assert.Equal(t, 1, logRepo.count())

	last := logRepo.last()
	require.NotNil(t, last)
	assert.Equal(t, domain.StatusError, last.Status)
	assert.Equal(t, "boom", last.ErrorMessage)
}

// TestEngine_OnLeadEvent_SkipsUnknownType verifies that automations whose type has no
// registered executor are silently skipped without creating a log entry.
func TestEngine_OnLeadEvent_SkipsUnknownType(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("auto-unk", "tenant-1", "funnel-1", "col-1", domain.TypeMoveFunnel)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	// Register an executor for a different type, not TypeMoveFunnel
	exec := &mockExecutor{}
	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoNote, exec)

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "col-1")
	require.NoError(t, err)

	assert.Equal(t, 0, exec.calls(), "executor for unrelated type must not be called")
	assert.Equal(t, 0, logRepo.count(), "no log should be created for skipped automation")
}

// TestEngine_OnLeadEvent_AsyncExecution verifies that async automation types (e.g. auto_message)
// are dispatched asynchronously and the log is eventually created.
func TestEngine_OnLeadEvent_AsyncExecution(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}

	a := makeAutomation("auto-async", "tenant-1", "funnel-1", "col-async", domain.TypeAutoMessage)
	require.NoError(t, autoRepo.Create(context.Background(), &a))

	exec := &mockExecutor{}
	engine := NewAutomationEngine(autoRepo, logRepo)
	engine.RegisterExecutor(domain.TypeAutoMessage, exec)

	err := engine.OnLeadEvent(context.Background(), "tenant-1", "lead-1", "col-async")
	require.NoError(t, err)

	// The goroutine should finish well within 1 second.
	ok := logRepo.waitForCount(1, 1*time.Second)
	assert.True(t, ok, "async execution log should be created within timeout")
	assert.Equal(t, 1, exec.calls())

	last := logRepo.last()
	require.NotNil(t, last)
	assert.Equal(t, domain.StatusSuccess, last.Status)
}
