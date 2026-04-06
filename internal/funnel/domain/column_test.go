package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewColumn_Valid(t *testing.T) {
	c, err := NewColumn("id-1", "funnel-1", "Novo", 0, ColumnTypeEntry, "#22c55e")
	require.NoError(t, err)
	assert.Equal(t, "Novo", c.Name)
	assert.Equal(t, ColumnTypeEntry, c.Type)
	assert.Equal(t, "#22c55e", c.Color)
}

func TestNewColumn_EmptyFunnelID(t *testing.T) {
	_, err := NewColumn("id-1", "", "Novo", 0, ColumnTypeEntry, "")
	assert.ErrorIs(t, err, ErrFunnelIDRequired)
}

func TestNewColumn_EmptyName(t *testing.T) {
	_, err := NewColumn("id-1", "funnel-1", "", 0, ColumnTypeEntry, "")
	assert.ErrorIs(t, err, ErrColumnNameRequired)
}

func TestNewColumn_NameTooLong(t *testing.T) {
	_, err := NewColumn("id-1", "funnel-1", strings.Repeat("a", MaxColumnNameLength+1), 0, ColumnTypeEntry, "")
	assert.ErrorIs(t, err, ErrColumnNameTooLong)
}

func TestNewColumn_InvalidType(t *testing.T) {
	_, err := NewColumn("id-1", "funnel-1", "Novo", 0, "invalid", "")
	assert.ErrorIs(t, err, ErrColumnTypeInvalid)
}

func TestNewColumn_DefaultColor(t *testing.T) {
	c, err := NewColumn("id-1", "funnel-1", "Novo", 0, ColumnTypeEntry, "")
	require.NoError(t, err)
	assert.Equal(t, "#3b82f6", c.Color)
}

func TestColumn_Update(t *testing.T) {
	c, _ := NewColumn("id-1", "funnel-1", "Old", 0, ColumnTypeEntry, "#000")
	err := c.Update("New", ColumnTypeIntermediate, "#fff")
	require.NoError(t, err)
	assert.Equal(t, "New", c.Name)
	assert.Equal(t, ColumnTypeIntermediate, c.Type)
	assert.Equal(t, "#fff", c.Color)
}

func TestAllColumnTypes(t *testing.T) {
	for _, ct := range []ColumnType{ColumnTypeEntry, ColumnTypeIntermediate, ColumnTypeWon, ColumnTypeLost} {
		_, err := NewColumn("id", "funnel", "Name", 0, ct, "")
		assert.NoError(t, err, "type %s should be valid", ct)
	}
}
