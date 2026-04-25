package application

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDiff_NoDifferences_ReturnsNil(t *testing.T) {
	before := map[string]any{"name": "ACME", "active": true}
	after := map[string]any{"name": "ACME", "active": true}

	diff := BuildDiff(before, after)
	assert.Nil(t, diff)
}

func TestBuildDiff_OneFieldChanged_ReturnsBeforeAfter(t *testing.T) {
	before := map[string]any{"name": "ACME"}
	after := map[string]any{"name": "Acme Holdings"}

	diff := BuildDiff(before, after)

	require.Contains(t, diff, "name")
	got, ok := diff["name"].(map[string]any)
	require.True(t, ok, "campo alterado deve ser map[string]any")
	assert.Equal(t, "ACME", got["antes"])
	assert.Equal(t, "Acme Holdings", got["depois"])
}

func TestBuildDiff_IgnoresUpdatedAtAndForbiddenKeys(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	before := map[string]any{
		"name":          "A",
		"updated_at":    now,
		"password":      "old",
		"password_hash": "h1",
		"token":         "t1",
		"secret":        "s1",
		"authorization": "Bearer 1",
		"hash":          "h",
	}
	after := map[string]any{
		"name":          "A",
		"updated_at":    later,
		"password":      "new",
		"password_hash": "h2",
		"token":         "t2",
		"secret":        "s2",
		"authorization": "Bearer 2",
		"hash":          "h-new",
	}

	diff := BuildDiff(before, after)
	assert.Nil(t, diff, "todas as alteracoes envolvem chaves filtradas")
}

func TestBuildDiff_ForbiddenKeysAreCaseInsensitive(t *testing.T) {
	before := map[string]any{
		"PASSWORD":      "x",
		"Password_Hash": "y",
		"Token":         "z",
	}
	after := map[string]any{
		"PASSWORD":      "x2",
		"Password_Hash": "y2",
		"Token":         "z2",
	}

	diff := BuildDiff(before, after)
	assert.Nil(t, diff)
}

func TestBuildDiff_FieldOnlyInBefore_TreatedAsRemoved(t *testing.T) {
	before := map[string]any{"role": "operador", "name": "A"}
	after := map[string]any{"name": "A"}

	diff := BuildDiff(before, after)

	require.Contains(t, diff, "role")
	got := diff["role"].(map[string]any)
	assert.Equal(t, "operador", got["antes"])
	assert.Nil(t, got["depois"])
}

func TestBuildDiff_FieldOnlyInAfter_TreatedAsAdded(t *testing.T) {
	before := map[string]any{"name": "A"}
	after := map[string]any{"role": "gestor", "name": "A"}

	diff := BuildDiff(before, after)

	require.Contains(t, diff, "role")
	got := diff["role"].(map[string]any)
	assert.Nil(t, got["antes"])
	assert.Equal(t, "gestor", got["depois"])
}

func TestBuildDiff_NilInputs_ReturnsNil(t *testing.T) {
	assert.Nil(t, BuildDiff(nil, nil))
	assert.Nil(t, BuildDiff(map[string]any{}, map[string]any{}))
}

func TestBuildDiff_MultipleChangesAndUnchangedMixed(t *testing.T) {
	before := map[string]any{
		"name":       "A",
		"role":       "operador",
		"active":     true,
		"updated_at": time.Now(),
	}
	after := map[string]any{
		"name":       "A",
		"role":       "gestor",
		"active":     false,
		"updated_at": time.Now().Add(time.Hour),
	}

	diff := BuildDiff(before, after)

	require.Contains(t, diff, "role")
	require.Contains(t, diff, "active")
	assert.NotContains(t, diff, "name")
	assert.NotContains(t, diff, "updated_at")
}
