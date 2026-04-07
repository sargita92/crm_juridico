package application

import (
	"context"
	"testing"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGroupUseCase_Success(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewCreateGroupUseCase(repo)

	out, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID:    "tenant-1",
		Name:        "Sales Team",
		Description: "Handles sales leads",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, out.ID)
	assert.Equal(t, "Sales Team", out.Name)
	assert.Equal(t, "Handles sales leads", out.Description)
	assert.Len(t, repo.groups, 1)
}

func TestCreateGroupUseCase_EmptyName(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewCreateGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID:    "tenant-1",
		Name:        "",
		Description: "No name",
	})

	assert.ErrorIs(t, err, domain.ErrGroupNameRequired)
	assert.Len(t, repo.groups, 0)
}

func TestCreateGroupUseCase_EmptyTenantID(t *testing.T) {
	repo := newMockGroupRepo()
	uc := NewCreateGroupUseCase(repo)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		TenantID:    "",
		Name:        "Some Group",
		Description: "",
	})

	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
	assert.Len(t, repo.groups, 0)
}
