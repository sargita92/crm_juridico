package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProduct_Valid(t *testing.T) {
	p, err := NewProduct("id-1", "Consultoria Trabalhista", "Desc", []string{"trabalhista", "CLT"})
	require.NoError(t, err)
	assert.Equal(t, "Consultoria Trabalhista", p.Name)
	assert.Equal(t, []string{"trabalhista", "CLT"}, p.Keywords)
	assert.True(t, p.Active)
}

func TestNewProduct_EmptyName(t *testing.T) {
	_, err := NewProduct("id-1", "", "", nil)
	assert.ErrorIs(t, err, ErrProductNameRequired)
}

func TestNewProduct_NameTooLong(t *testing.T) {
	_, err := NewProduct("id-1", strings.Repeat("a", MaxProductNameLength+1), "", nil)
	assert.ErrorIs(t, err, ErrProductNameTooLong)
}

func TestNewProduct_NilKeywords(t *testing.T) {
	p, err := NewProduct("id-1", "Produto", "", nil)
	require.NoError(t, err)
	assert.Empty(t, p.Keywords)
}

func TestProduct_Update(t *testing.T) {
	p, _ := NewProduct("id-1", "Old", "old desc", nil)
	err := p.Update("New", "new desc", []string{"kw1"})
	require.NoError(t, err)
	assert.Equal(t, "New", p.Name)
	assert.Equal(t, "new desc", p.Description)
	assert.Equal(t, []string{"kw1"}, p.Keywords)
}

func TestProduct_Update_EmptyName(t *testing.T) {
	p, _ := NewProduct("id-1", "Old", "", nil)
	assert.ErrorIs(t, p.Update("", "", nil), ErrProductNameRequired)
}

func TestProduct_ActivateDeactivate(t *testing.T) {
	p, _ := NewProduct("id-1", "Test", "", nil)
	p.Deactivate()
	assert.False(t, p.Active)
	p.Activate()
	assert.True(t, p.Active)
}

func TestProduct_MatchesKeyword(t *testing.T) {
	p, _ := NewProduct("id-1", "Trabalhista", "", []string{"trabalhista", "CLT", "rescisao"})
	assert.True(t, p.MatchesText("Preciso de ajuda com questao trabalhista"))
	assert.True(t, p.MatchesText("Meu caso e sobre CLT"))
	assert.True(t, p.MatchesText("TRABALHISTA urgente"))
	assert.False(t, p.MatchesText("Quero falar sobre aposentadoria"))
}

func TestProduct_MatchesKeyword_NoKeywords(t *testing.T) {
	p, _ := NewProduct("id-1", "Produto", "", nil)
	assert.False(t, p.MatchesText("qualquer coisa"))
}
