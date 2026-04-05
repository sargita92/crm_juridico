package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
