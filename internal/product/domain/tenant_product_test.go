package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenantProduct_Success(t *testing.T) {
	tp, err := NewTenantProduct("id-1", "tenant-1", "prod-1")
	require.NoError(t, err)
	assert.Equal(t, "id-1", tp.ID)
	assert.Equal(t, "tenant-1", tp.TenantID)
	assert.Equal(t, "prod-1", tp.ProductID)
	assert.False(t, tp.CreatedAt.IsZero())
}

func TestNewTenantProduct_EmptyTenantID(t *testing.T) {
	_, err := NewTenantProduct("id", "", "prod-1")
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewTenantProduct_EmptyProductID(t *testing.T) {
	_, err := NewTenantProduct("id", "tenant-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID")
}
