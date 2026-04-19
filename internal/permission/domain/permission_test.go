package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

func TestNewPermission_PaymentsView(t *testing.T) {
	p, err := domain.NewPermission("id-1", "tenant-1", "group-1", "", "payments", "view")
	require.NoError(t, err)
	assert.Equal(t, "payments", p.Resource)
	assert.Equal(t, "view", p.Action)
	assert.True(t, p.IsGroupPermission())
}

func TestNewPermission_PaymentsInvalidAction(t *testing.T) {
	_, err := domain.NewPermission("id-1", "tenant-1", "group-1", "", "payments", "manage")
	assert.ErrorIs(t, err, domain.ErrInvalidAction)
}

func TestNewPermission_UnknownResource(t *testing.T) {
	_, err := domain.NewPermission("id-1", "tenant-1", "group-1", "", "lixo", "view")
	assert.ErrorIs(t, err, domain.ErrInvalidResource)
}

func TestNewPermission_XORViolation(t *testing.T) {
	_, err := domain.NewPermission("id-1", "tenant-1", "group-1", "user-1", "payments", "view")
	assert.ErrorIs(t, err, domain.ErrPermissionXOR)
}
