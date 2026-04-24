package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilter_Normalize_DefaultsPageSize(t *testing.T) {
	f := Filter{}
	require.NoError(t, f.Normalize())
	assert.Equal(t, DefaultPageSize, f.PageSize)
	assert.Equal(t, 1, f.Page)
}

func TestFilter_Normalize_DefaultsPageSizeWhenNegative(t *testing.T) {
	f := Filter{PageSize: -5}
	require.NoError(t, f.Normalize())
	assert.Equal(t, DefaultPageSize, f.PageSize)
}

// S3-C18 (decisao 1) — clamp para o maximo, sem erro.
func TestFilter_Normalize_ClampsPageSizeToMax(t *testing.T) {
	f := Filter{PageSize: 999}
	require.NoError(t, f.Normalize())
	assert.Equal(t, MaxPageSize, f.PageSize)
}

func TestFilter_Normalize_KeepsPageSizeInRange(t *testing.T) {
	for _, ps := range []int{10, 25, 50, 100} {
		f := Filter{PageSize: ps}
		require.NoError(t, f.Normalize())
		assert.Equal(t, ps, f.PageSize)
	}
}

// S3-C19 — page <= 0 normaliza para 1.
func TestFilter_Normalize_PageZeroBecomesOne(t *testing.T) {
	f := Filter{Page: 0}
	require.NoError(t, f.Normalize())
	assert.Equal(t, 1, f.Page)
}

func TestFilter_Normalize_NegativePageBecomesOne(t *testing.T) {
	f := Filter{Page: -3}
	require.NoError(t, f.Normalize())
	assert.Equal(t, 1, f.Page)
}

func TestFilter_Normalize_KeepsPositivePage(t *testing.T) {
	f := Filter{Page: 5}
	require.NoError(t, f.Normalize())
	assert.Equal(t, 5, f.Page)
}

// S3-C17 — from > to retorna ErrInvalidPeriod.
func TestFilter_Normalize_RejectsFromAfterTo(t *testing.T) {
	from := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	f := Filter{From: &from, To: &to}
	err := f.Normalize()
	assert.ErrorIs(t, err, ErrInvalidPeriod)
}

func TestFilter_Normalize_AllowsFromEqualTo(t *testing.T) {
	t0 := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	f := Filter{From: &t0, To: &t0}
	assert.NoError(t, f.Normalize())
}

func TestFilter_Normalize_AllowsOnlyFrom(t *testing.T) {
	from := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	f := Filter{From: &from}
	assert.NoError(t, f.Normalize())
}

func TestFilter_Normalize_AllowsOnlyTo(t *testing.T) {
	to := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	f := Filter{To: &to}
	assert.NoError(t, f.Normalize())
}

// S3-C20 (decisao 2) — action invalida retorna ErrInvalidAction.
func TestFilter_Normalize_RejectsInvalidAction(t *testing.T) {
	a := Action("hack.invalida")
	f := Filter{Action: &a}
	err := f.Normalize()
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestFilter_Normalize_AcceptsValidAction(t *testing.T) {
	a := ActionLoginSuccess
	f := Filter{Action: &a}
	assert.NoError(t, f.Normalize())
}

func TestFilter_Normalize_NilActionIsAllowed(t *testing.T) {
	f := Filter{}
	assert.NoError(t, f.Normalize())
}

// Offset usa os valores ja normalizados.
func TestFilter_Offset(t *testing.T) {
	cases := []struct {
		page     int
		pageSize int
		want     int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 25, 50},
		{5, 100, 400},
	}
	for _, c := range cases {
		f := Filter{Page: c.page, PageSize: c.pageSize}
		assert.Equal(t, c.want, f.Offset(), "page=%d size=%d", c.page, c.pageSize)
	}
}
