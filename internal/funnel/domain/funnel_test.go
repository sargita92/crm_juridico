package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFunnel_Valid(t *testing.T) {
	f, err := NewFunnel("id-1", "tenant-1", "Pipeline", "descricao")
	require.NoError(t, err)
	assert.Equal(t, "Pipeline", f.Name)
	assert.True(t, f.Active)
	assert.False(t, f.IsDefault)
}

func TestNewFunnel_EmptyTenant(t *testing.T) {
	_, err := NewFunnel("id-1", "", "Pipeline", "")
	assert.ErrorIs(t, err, ErrTenantIDRequired)
}

func TestNewFunnel_EmptyName(t *testing.T) {
	_, err := NewFunnel("id-1", "tenant-1", "", "")
	assert.ErrorIs(t, err, ErrFunnelNameRequired)
}

func TestNewFunnel_NameTooLong(t *testing.T) {
	_, err := NewFunnel("id-1", "tenant-1", strings.Repeat("a", MaxFunnelNameLength+1), "")
	assert.ErrorIs(t, err, ErrFunnelNameTooLong)
}

func TestFunnel_Update(t *testing.T) {
	f, _ := NewFunnel("id-1", "tenant-1", "Old", "old desc")
	err := f.Update("New", "new desc")
	require.NoError(t, err)
	assert.Equal(t, "New", f.Name)
	assert.Equal(t, "new desc", f.Description)
}

func TestFunnel_Update_EmptyName(t *testing.T) {
	f, _ := NewFunnel("id-1", "tenant-1", "Old", "")
	assert.ErrorIs(t, f.Update("", ""), ErrFunnelNameRequired)
}

func TestFunnel_ActivateDeactivate(t *testing.T) {
	f, _ := NewFunnel("id-1", "tenant-1", "Test", "")
	f.Deactivate()
	assert.False(t, f.Active)
	f.Activate()
	assert.True(t, f.Active)
}

func TestFunnel_SetDefault(t *testing.T) {
	f, _ := NewFunnel("id-1", "tenant-1", "Test", "")
	f.SetDefault()
	assert.True(t, f.IsDefault)
}
