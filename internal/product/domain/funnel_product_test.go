package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFunnelProduct_Valid(t *testing.T) {
	fp, err := NewFunnelProduct("id-1", "funnel-1", "product-1", 10)
	require.NoError(t, err)
	assert.Equal(t, "funnel-1", fp.FunnelID)
	assert.Equal(t, "product-1", fp.ProductID)
	assert.Equal(t, 10, fp.Priority)
}

func TestNewFunnelProduct_EmptyFunnelID(t *testing.T) {
	_, err := NewFunnelProduct("id-1", "", "product-1", 10)
	assert.Error(t, err)
}

func TestNewFunnelProduct_EmptyProductID(t *testing.T) {
	_, err := NewFunnelProduct("id-1", "funnel-1", "", 10)
	assert.Error(t, err)
}

func TestNewFunnelProduct_ZeroPriority(t *testing.T) {
	fp, err := NewFunnelProduct("id-1", "funnel-1", "product-1", 0)
	require.NoError(t, err)
	assert.Equal(t, 1, fp.Priority, "zero priority should default to 1")
}
