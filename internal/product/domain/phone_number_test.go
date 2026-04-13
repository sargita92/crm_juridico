package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProductPhoneNumber_Success(t *testing.T) {
	p, err := NewProductPhoneNumber("id-1", "prod-1", "+5511999999999")
	require.NoError(t, err)
	assert.Equal(t, "id-1", p.ID)
	assert.Equal(t, "prod-1", p.ProductID)
	assert.Equal(t, "+5511999999999", p.PhoneNumber)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestNewProductPhoneNumber_EmptyProductID(t *testing.T) {
	_, err := NewProductPhoneNumber("id", "", "+5511999999999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "product ID")
}

func TestNewProductPhoneNumber_EmptyPhoneNumber(t *testing.T) {
	_, err := NewProductPhoneNumber("id", "prod-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "phone number")
}
