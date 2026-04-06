package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

func TestCreateColumn_Success(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	output, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1",
		FunnelID: f.ID,
		Name:     "Nova Coluna",
		Type:     domain.ColumnTypeIntermediate,
		Color:    "#ff0000",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Nova Coluna", output.Name)
	assert.Equal(t, 0, output.OrderIndex)
	assert.Equal(t, "#ff0000", output.Color)
}

func TestCreateColumn_OrderIncrement(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		Name: "Col 1", Type: domain.ColumnTypeEntry, Color: "",
	})
	require.NoError(t, err)

	output, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		Name: "Col 2", Type: domain.ColumnTypeIntermediate, Color: "",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, output.OrderIndex)
}

func TestCreateColumn_LimitReached(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	for i := 0; i < domain.MaxColumnsPerFunnel; i++ {
		col, _ := domain.NewColumn(uuid.New().String(), f.ID, "Col", i, domain.ColumnTypeIntermediate, "")
		_ = columnRepo.Create(context.Background(), col)
	}

	_, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		Name: "Extra", Type: domain.ColumnTypeIntermediate,
	})
	assert.ErrorIs(t, err, domain.ErrColumnLimitReached)
}

func TestCreateColumn_FunnelNotFound(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	_, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1", FunnelID: "nope",
		Name: "Col", Type: domain.ColumnTypeEntry,
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}

func TestCreateColumn_InactiveFunnel(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	f.Deactivate()
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "tenant-1", FunnelID: f.ID,
		Name: "Col", Type: domain.ColumnTypeEntry,
	})
	assert.ErrorIs(t, err, domain.ErrFunnelInactive)
}

func TestCreateColumn_WrongTenant(t *testing.T) {
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	uc := NewCreateColumnUseCase(funnelRepo, columnRepo)

	f, _ := domain.NewFunnel(uuid.New().String(), "tenant-1", "Pipeline", "")
	_ = funnelRepo.Create(context.Background(), f)

	_, err := uc.Execute(context.Background(), CreateColumnInput{
		TenantID: "other", FunnelID: f.ID,
		Name: "Col", Type: domain.ColumnTypeEntry,
	})
	assert.ErrorIs(t, err, domain.ErrFunnelNotFound)
}
