package infrastructure

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestGormScoringConfig_PersistsHumanoFields(t *testing.T) {
	repo, db := setupScoringRepo(t)

	ctx := context.Background()
	specialistID := createTestSpecialistForStep(t, db)

	sc := &domain.ScoringConfig{
		ID:                 uuid.New().String(),
		SpecialistID:       specialistID,
		Threshold:          80,
		ThresholdHumanoMin: 50,
		HumanColumnID:      "col-h",
		CrossSellColumnID:  "col-cs",
	}

	require.NoError(t, repo.CreateOrUpdate(ctx, sc))

	got, err := repo.FindBySpecialistID(ctx, specialistID)
	require.NoError(t, err)
	assert.Equal(t, 50, got.ThresholdHumanoMin)
	assert.Equal(t, "col-h", got.HumanColumnID)
	assert.Equal(t, "col-cs", got.CrossSellColumnID)
}
