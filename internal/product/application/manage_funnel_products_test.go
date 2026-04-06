package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

func TestManageFunnelProducts_LinkSuccess(t *testing.T) {
	fpRepo := newMockFunnelProductRepo()
	uc := NewManageFunnelProductsUseCase(fpRepo)

	err := uc.Link(context.Background(), LinkFunnelProductInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
		Priority:  10,
	})

	require.NoError(t, err)
	links, err := fpRepo.FindByFunnelID(context.Background(), "funnel-1")
	require.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, "prod-1", links[0].ProductID)
	assert.Equal(t, 10, links[0].Priority)
}

func TestManageFunnelProducts_LinkDuplicate_Error(t *testing.T) {
	fpRepo := newMockFunnelProductRepo()
	uc := NewManageFunnelProductsUseCase(fpRepo)

	input := LinkFunnelProductInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
		Priority:  10,
	}

	require.NoError(t, uc.Link(context.Background(), input))

	err := uc.Link(context.Background(), input)
	assert.ErrorIs(t, err, domain.ErrFunnelProductAlreadyExists)
}

func TestManageFunnelProducts_UnlinkSuccess(t *testing.T) {
	fpRepo := newMockFunnelProductRepo()
	uc := NewManageFunnelProductsUseCase(fpRepo)

	require.NoError(t, uc.Link(context.Background(), LinkFunnelProductInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
		Priority:  5,
	}))

	err := uc.Unlink(context.Background(), UnlinkFunnelProductInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
	})

	require.NoError(t, err)
	links, err := fpRepo.FindByFunnelID(context.Background(), "funnel-1")
	require.NoError(t, err)
	assert.Len(t, links, 0)
}

func TestManageFunnelProducts_UpdatePriority(t *testing.T) {
	fpRepo := newMockFunnelProductRepo()
	uc := NewManageFunnelProductsUseCase(fpRepo)

	require.NoError(t, uc.Link(context.Background(), LinkFunnelProductInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
		Priority:  5,
	}))

	err := uc.UpdatePriority(context.Background(), UpdateFunnelProductPriorityInput{
		FunnelID:  "funnel-1",
		ProductID: "prod-1",
		Priority:  99,
	})

	require.NoError(t, err)
	links, err := fpRepo.FindByFunnelID(context.Background(), "funnel-1")
	require.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, 99, links[0].Priority)
}
