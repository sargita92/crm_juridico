package application

import (
	"context"
	"testing"
	"time"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAutomation_Success(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	out, err := uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "tenant-1",
		FunnelID: "funnel-1",
		ColumnID: "col-1",
		Type:     string(domain.TypeMoveFunnel),
		Config:   map[string]interface{}{"target_funnel": "funnel-2"},
		Priority: 1,
	})

	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "tenant-1", out.TenantID)
	assert.Equal(t, "funnel-1", out.FunnelID)
	assert.Equal(t, "col-1", out.ColumnID)
	assert.Equal(t, string(domain.TypeMoveFunnel), out.Type)
	assert.True(t, out.Active)
	assert.Equal(t, 1, out.Priority)
	assert.NotEmpty(t, out.ID)
}

func TestCreateAutomation_InvalidType(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	out, err := uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "tenant-1",
		FunnelID: "funnel-1",
		Type:     "invalid_type",
	})

	assert.Nil(t, out)
	assert.ErrorIs(t, err, domain.ErrInvalidType)
}

func TestListByFunnel(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	// Create two automations for same funnel, one for a different funnel
	_, err := uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "t1", FunnelID: "f1", Type: string(domain.TypeAutoNote), Priority: 1,
	})
	require.NoError(t, err)

	_, err = uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "t1", FunnelID: "f1", Type: string(domain.TypeMoveFunnel), Priority: 2,
	})
	require.NoError(t, err)

	_, err = uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "t1", FunnelID: "f2", Type: string(domain.TypeAutoNote), Priority: 1,
	})
	require.NoError(t, err)

	out, err := uc.ListByFunnel(context.Background(), "t1", "f1")
	require.NoError(t, err)
	assert.Len(t, out, 2)
	for _, a := range out {
		assert.Equal(t, "f1", a.FunnelID)
		assert.Equal(t, "t1", a.TenantID)
	}
}

func TestToggle(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	created, err := uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "t1", FunnelID: "f1", Type: string(domain.TypeAutoNote),
	})
	require.NoError(t, err)
	assert.True(t, created.Active)

	// Toggle off
	err = uc.Toggle(context.Background(), created.ID)
	require.NoError(t, err)

	fetched, err := uc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.False(t, fetched.Active)

	// Toggle on
	err = uc.Toggle(context.Background(), created.ID)
	require.NoError(t, err)

	fetched, err = uc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	assert.True(t, fetched.Active)
}

func TestDelete(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	created, err := uc.Create(context.Background(), CreateAutomationInput{
		TenantID: "t1", FunnelID: "f1", Type: string(domain.TypeAutoNote),
	})
	require.NoError(t, err)

	err = uc.Delete(context.Background(), created.ID)
	require.NoError(t, err)

	_, err = uc.GetByID(context.Background(), created.ID)
	assert.ErrorIs(t, err, domain.ErrAutomationNotFound)
}

func TestGetLogs(t *testing.T) {
	autoRepo := &mockAutoRepo{}
	logRepo := &mockLogRepo{}
	uc := NewCRUDUseCase(autoRepo, logRepo)

	// Seed logs manually into the mock
	logRepo.logs = append(logRepo.logs,
		&domain.ExecutionLog{
			ID:           "log-1",
			AutomationID: "auto-1",
			LeadID:       "lead-1",
			TenantID:     "t1",
			Status:       domain.StatusSuccess,
			ExecutedAt:   time.Now(),
		},
		&domain.ExecutionLog{
			ID:           "log-2",
			AutomationID: "auto-1",
			LeadID:       "lead-2",
			TenantID:     "t1",
			Status:       domain.StatusError,
			ErrorMessage: "something failed",
			ExecutedAt:   time.Now(),
		},
	)

	out, err := uc.GetLogs(context.Background(), "auto-1", 10, 0)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "log-1", out[0].ID)
	assert.Equal(t, "log-2", out[1].ID)
	assert.Equal(t, "something failed", out[1].ErrorMessage)
}
