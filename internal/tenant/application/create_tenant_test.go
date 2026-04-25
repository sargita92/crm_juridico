package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func TestCreateTenantUseCase_Success(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewCreateTenantUseCase(repo)

	output, err := uc.Execute(context.Background(), CreateTenantInput{
		Name:     "Escritório ABC",
		Type:     "PJ",
		Document: "12.345.678/0001-90",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Escritório ABC", output.Name)
	assert.Equal(t, "PJ", output.Type)
	assert.Equal(t, "12.345.678/0001-90", output.Document)
	assert.Equal(t, "active", output.Status)
}

func TestCreateTenantUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewCreateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateTenantInput{
		Name:     "",
		Type:     "PJ",
		Document: "12.345.678/0001-90",
	})

	assert.ErrorIs(t, err, domain.ErrTenantNameRequired)
}

func TestCreateTenantUseCase_InvalidType_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewCreateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateTenantInput{
		Name:     "Escritório",
		Type:     "INVALID",
		Document: "12345",
	})

	assert.ErrorIs(t, err, domain.ErrInvalidTenantType)
}

func TestCreateTenantUseCase_DuplicateDocument_ReturnsError(t *testing.T) {
	repo := newMockTenantRepo()
	uc := NewCreateTenantUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateTenantInput{
		Name:     "Escritório A",
		Type:     "PJ",
		Document: "12.345.678/0001-90",
	})
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), CreateTenantInput{
		Name:     "Escritório B",
		Type:     "PJ",
		Document: "12.345.678/0001-90",
	})
	assert.ErrorIs(t, err, domain.ErrTenantDocumentExists)
}

// ctxWithAdminClaims monta um context com TokenClaims (admin) usado pelas
// UCs para popular ActorEmail e UserID no audit publisher.
func ctxWithAdminClaims(userID, email string) context.Context {
	return middleware.SetClaimsForTest(context.Background(), &authdomain.TokenClaims{
		UserID: userID,
		Email:  email,
		Role:   authdomain.UserRoleAdmin,
	})
}

func TestCreateTenantUseCase_PublishesAuditOnSuccess(t *testing.T) {
	repo := newMockTenantRepo()
	spy := &spyAuditPublisher{}
	uc := NewCreateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	ctx := ctxWithAdminClaims("admin-1", "admin@crm.com")

	out, err := uc.Execute(ctx, CreateTenantInput{
		Name: "Escritório ABC", Type: "PJ", Document: "12345",
	})
	require.NoError(t, err)
	require.Len(t, spy.calls, 1)
	call := spy.calls[0]
	assert.Equal(t, auditdomain.ActionTenantCreated, call.Action)
	assert.Equal(t, "tenant", call.Entity)
	require.NotNil(t, call.EntityID)
	assert.Equal(t, out.ID, *call.EntityID)
	require.NotNil(t, call.TenantID)
	assert.Equal(t, out.ID, *call.TenantID)
	assert.Equal(t, "admin@crm.com", call.ActorEmail)
	require.NotNil(t, call.UserID)
	assert.Equal(t, "admin-1", *call.UserID)
}

func TestCreateTenantUseCase_DoesNotPublishWhenRepoFails(t *testing.T) {
	repo := newMockTenantRepo()
	repo.createErr = errors.New("db down")
	spy := &spyAuditPublisher{}
	uc := NewCreateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	_, err := uc.Execute(ctxWithAdminClaims("admin-1", "admin@crm.com"), CreateTenantInput{
		Name: "X", Type: "PJ", Document: "1",
	})
	require.Error(t, err)
	assert.Empty(t, spy.calls)
}

func TestCreateTenantUseCase_PublisherErrorDoesNotAbort(t *testing.T) {
	repo := newMockTenantRepo()
	spy := &spyAuditPublisher{publishErr: errors.New("audit insert failed")}
	uc := NewCreateTenantUseCase(repo)
	uc.SetAuditPublisher(spy)

	out, err := uc.Execute(ctxWithAdminClaims("a", "a@x.com"), CreateTenantInput{
		Name: "X", Type: "PJ", Document: "1",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
}
