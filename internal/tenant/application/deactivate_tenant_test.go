package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestDeactivateTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusInactive, found.Status)
}

func TestDeactivateTenantUseCase_NotFound(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrTenantNotFound)
}

func TestDeactivateTenantUseCase_AlreadyInactive(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Deactivate()
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	assert.ErrorIs(t, err, domain.ErrTenantInactive)
}

func TestDeactivateTenantUseCase_FromBlocked(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "Escritório", domain.TenantTypePJ, "111")
	_ = tenant.Block("Motivo")
	repo.addTenant(tenant)

	uc := NewDeactivateTenantUseCase(repo)

	err := uc.Execute(context.Background(), "id-1")
	require.NoError(t, err)

	found, _ := repo.FindByID(context.Background(), "id-1")
	assert.Equal(t, domain.TenantStatusInactive, found.Status)
}

func TestDeactivateTenantUseCase_PublishesAuditOnSuccess(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{}
	uc := NewDeactivateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("admin-1", "admin@crm.com"), "id-1")
	require.NoError(t, err)

	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionTenantDeactivated, call.Action)
	assert.Equal(t, "tenant", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, "id-1", *call.EntityID)
	require.NotNil(t, call.TenantID)
	assert.Equal(t, "id-1", *call.TenantID)
	assert.Equal(t, "admin@crm.com", call.ActorEmail)
}

func TestDeactivateTenantUseCase_RepoFailure_DoesNotPublish(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)
	repo.updateErr = errors.New("db down")

	spy := &spyAuditPublisher{}
	uc := NewDeactivateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), "id-1")
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestDeactivateTenantUseCase_PublisherError_DoesNotAbort(t *testing.T) {
	repo := newMockTenantRepo()
	tenant, _ := domain.NewTenant("id-1", "X", domain.TenantTypePJ, "111")
	repo.addTenant(tenant)

	spy := &spyAuditPublisher{publishErr: errors.New("audit fail")}
	uc := NewDeactivateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), "id-1")
	require.NoError(t, err)
}
