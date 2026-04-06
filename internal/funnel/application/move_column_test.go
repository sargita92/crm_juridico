package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func setupMoveColumnTest(t *testing.T) (*MoveColumnUseCase, *mockFunnelRepo, *mockColumnRepo, *domain.Funnel, []*domain.Column) {
	t.Helper()
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewMoveColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	cols := make([]*domain.Column, 3)
	for i := 0; i < 3; i++ {
		cols[i], _ = domain.NewColumn(uuid.New().String(), f.ID, "Col", i, domain.ColumnTypeIntermediate, "")
		_ = columnRepo.Create(context.Background(), cols[i])
	}

	return uc, funnelRepo, columnRepo, f, cols
}

func TestMoveColumn_Down(t *testing.T) {
	uc, _, columnRepo, f, cols := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: cols[0].ID, Direction: "down",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, columnRepo.columns[cols[0].ID].OrderIndex)
	assert.Equal(t, 0, columnRepo.columns[cols[1].ID].OrderIndex)
}

func TestMoveColumn_Up(t *testing.T) {
	uc, _, columnRepo, f, cols := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: cols[2].ID, Direction: "up",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, columnRepo.columns[cols[2].ID].OrderIndex)
	assert.Equal(t, 2, columnRepo.columns[cols[1].ID].OrderIndex)
}

func TestMoveColumn_CannotMoveFirstUp(t *testing.T) {
	uc, _, _, f, cols := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: cols[0].ID, Direction: "up",
	})
	assert.ErrorIs(t, err, ErrCannotMoveColumn)
}

func TestMoveColumn_CannotMoveLastDown(t *testing.T) {
	uc, _, _, f, cols := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: cols[2].ID, Direction: "down",
	})
	assert.ErrorIs(t, err, ErrCannotMoveColumn)
}

func TestMoveColumn_InvalidDirection(t *testing.T) {
	uc, _, _, f, cols := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: cols[0].ID, Direction: "left",
	})
	assert.ErrorIs(t, err, ErrInvalidDirection)
}

func TestMoveColumn_FunnelNotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewMoveColumnUseCase(funnelRepo, columnRepo)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: "nope",
		ColumnID: "col", Direction: "up",
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestMoveColumn_ColumnNotFound(t *testing.T) {
	uc, _, _, f, _ := setupMoveColumnTest(t)

	err := uc.Execute(context.Background(), MoveColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		ColumnID: "nonexistent", Direction: "up",
	})
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}
