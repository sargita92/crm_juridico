package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedUserTenant associa user↔tenant com a flag is_owner.
func seedUserTenant(t *testing.T, db *gorm.DB, userID, tenantID string, isOwner bool) {
	t.Helper()
	err := db.Exec(`INSERT INTO user_tenants (user_id, tenant_id, is_owner)
		VALUES (?, ?, ?)`, userID, tenantID, isOwner).Error
	require.NoError(t, err)
}

func TestOperators_ReturnsOnlyNonOwnersOrdered(t *testing.T) {
	_, db := setupStatsRepo(t)
	lister := NewGormUserLookup(db)
	tenantID := seedTenant(t, db)

	owner := seedUser(t, db, "Zelia Owner")
	op2 := seedUser(t, db, "Bruno")
	op1 := seedUser(t, db, "Ana")
	seedUserTenant(t, db, owner, tenantID, true)
	seedUserTenant(t, db, op2, tenantID, false)
	seedUserTenant(t, db, op1, tenantID, false)

	// operador de OUTRO tenant não deve aparecer (isolamento)
	otherTenant := seedTenant(t, db)
	otherOp := seedUser(t, db, "Fora")
	seedUserTenant(t, db, otherOp, otherTenant, false)

	ops, err := lister.Operators(context.Background(), tenantID)
	require.NoError(t, err)
	require.Len(t, ops, 2)
	assert.Equal(t, op1, ops[0].ID) // ordenado por nome: Ana antes de Bruno
	assert.Equal(t, "Ana", ops[0].Name)
	assert.Equal(t, op2, ops[1].ID)
	assert.Equal(t, "Bruno", ops[1].Name)
}

func TestOperators_EmptyTenant(t *testing.T) {
	_, db := setupStatsRepo(t)
	lister := NewGormUserLookup(db)
	tenantID := seedTenant(t, db)

	ops, err := lister.Operators(context.Background(), tenantID)
	require.NoError(t, err)
	assert.NotNil(t, ops, "deve retornar slice vazio (não nil) p/ o template iterar")
	assert.Empty(t, ops)
}
