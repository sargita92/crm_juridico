package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestCalculateOutcome_BinaryWhenHumanoMinZero(t *testing.T) {
	sc := &domain.ScoringConfig{Threshold: 80, ThresholdHumanoMin: 0}
	assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 80))
	assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 100))
	assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 0))
	assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 79))
}

func TestCalculateOutcome_ThreeZonesWhenHumanoMinSet(t *testing.T) {
	sc := &domain.ScoringConfig{Threshold: 80, ThresholdHumanoMin: 50}
	assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 80))
	assert.Equal(t, domain.OutcomeAprovado, domain.CalculateOutcome(sc, 100))
	assert.Equal(t, domain.OutcomeHumano, domain.CalculateOutcome(sc, 50))
	assert.Equal(t, domain.OutcomeHumano, domain.CalculateOutcome(sc, 79))
	assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 49))
	assert.Equal(t, domain.OutcomeReprovado, domain.CalculateOutcome(sc, 0))
}
