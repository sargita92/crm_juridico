package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSpecialistProduct_WithValidData_ReturnsProduct(t *testing.T) {
	sp, err := NewSpecialistProduct("id-1", "spec-1", "prod-1")

	require.NoError(t, err)
	assert.Equal(t, "id-1", sp.ID)
	assert.Equal(t, "spec-1", sp.SpecialistID)
	assert.Equal(t, "prod-1", sp.ProductID)
	assert.False(t, sp.CreatedAt.IsZero())
}

func TestNewSpecialistProduct_EmptySpecialistID_ReturnsError(t *testing.T) {
	_, err := NewSpecialistProduct("id-1", "", "prod-1")

	assert.ErrorIs(t, err, ErrSpecialistIDRequired)
}

func TestNewSpecialistProduct_EmptyProductID_ReturnsError(t *testing.T) {
	_, err := NewSpecialistProduct("id-1", "spec-1", "")

	assert.ErrorIs(t, err, ErrProductIDRequired)
}
