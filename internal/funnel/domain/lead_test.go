package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLead_Valid(t *testing.T) {
	l, err := NewLead("id-1", "tenant-1", "funnel-1", "col-1", "contact-1", "conv-1")
	require.NoError(t, err)
	assert.Equal(t, LeadStatusOpen, l.Status)
	assert.Equal(t, 0, l.Score)
}

func TestNewLead_MissingFields(t *testing.T) {
	tests := []struct {
		name   string
		tenant string
		funnel string
		col    string
		contact string
		conv   string
		err    error
	}{
		{"no tenant", "", "f", "c", "ct", "cv", ErrTenantIDRequired},
		{"no funnel", "t", "", "c", "ct", "cv", ErrFunnelIDRequired},
		{"no column", "t", "f", "", "ct", "cv", ErrColumnIDRequired},
		{"no contact", "t", "f", "c", "", "cv", ErrContactIDRequired},
		{"no conv", "t", "f", "c", "ct", "", ErrConversationIDRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewLead("id", tc.tenant, tc.funnel, tc.col, tc.contact, tc.conv)
			assert.ErrorIs(t, err, tc.err)
		})
	}
}

func TestLead_MoveTo_Intermediate(t *testing.T) {
	l, _ := NewLead("id", "t", "f", "col-1", "ct", "cv")
	l.MoveTo("col-2", ColumnTypeIntermediate)
	assert.Equal(t, "col-2", l.ColumnID)
	assert.Equal(t, LeadStatusOpen, l.Status)
}

func TestLead_MoveTo_Won(t *testing.T) {
	l, _ := NewLead("id", "t", "f", "col-1", "ct", "cv")
	l.MoveTo("col-won", ColumnTypeWon)
	assert.Equal(t, LeadStatusWon, l.Status)
}

func TestLead_MoveTo_Lost(t *testing.T) {
	l, _ := NewLead("id", "t", "f", "col-1", "ct", "cv")
	l.MoveTo("col-lost", ColumnTypeLost)
	assert.Equal(t, LeadStatusLost, l.Status)
}

func TestLead_MoveTo_ReopensFromWon(t *testing.T) {
	l, _ := NewLead("id", "t", "f", "col-1", "ct", "cv")
	l.MoveTo("col-won", ColumnTypeWon)
	assert.Equal(t, LeadStatusWon, l.Status)
	l.MoveTo("col-2", ColumnTypeIntermediate)
	assert.Equal(t, LeadStatusOpen, l.Status)
}

func TestLead_UpdateScore(t *testing.T) {
	l, _ := NewLead("id", "t", "f", "c", "ct", "cv")
	l.UpdateScore(80)
	assert.Equal(t, 80, l.Score)
}
