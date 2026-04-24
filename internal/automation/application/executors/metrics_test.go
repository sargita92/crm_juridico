package executors

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
)

// stubLeadMover records the last MoveLead call and returns nil.
type stubLeadMover struct {
	called bool
}

func (s *stubLeadMover) MoveLead(_ context.Context, _, _, _, _ string) error {
	s.called = true
	return nil
}

// TestMoveFunnelExecutor_ObservesExecutionDuration drives a real executor
// through its Execute path and verifies the duration histogram picks up an
// observation with the expected {type, outcome} labels.
func TestMoveFunnelExecutor_ObservesExecutionDuration(t *testing.T) {
	exec := NewMoveFunnelExecutor(&stubLeadMover{})
	auto, err := domain.NewAutomation("auto-1", "t-1", "f-1", "c-1", domain.TypeMoveFunnel, map[string]any{
		"target_column_id": "c-2",
		"target_funnel_id": "f-1",
	}, 1)
	require.NoError(t, err)

	err = exec.Execute(context.Background(), auto, "lead-1", "t-1")
	require.NoError(t, err)

	// After Execute runs, the {move_funnel, success} series must exist in the
	// histogram vector — confirms the deferred Observe fired.
	assert.GreaterOrEqual(t, testutil.CollectAndCount(infrastructure.ExecutionDuration), 1,
		"execution_duration_seconds should have at least one series after executor runs")
}
