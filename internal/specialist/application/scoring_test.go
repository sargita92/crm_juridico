package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- GetScoringUseCase ---

func TestGetScoringUseCase_WithStepsAndConfig(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewGetScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 30)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Pergunta 2", domain.StepDataTypeNumber, false, 20)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	config, _ := domain.NewScoringConfig("cfg-1", "spec-1", 40)
	scoringRepo.addScoringConfig(config)

	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, 40, output.Threshold)
	assert.Equal(t, 50, output.TotalPossible)
	assert.InDelta(t, 80.0, output.Percentage, 0.01)
	assert.Len(t, output.Steps, 2)
}

func TestGetScoringUseCase_WithStepsNoConfig(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewGetScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 10)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Pergunta 2", domain.StepDataTypeDate, false, 15)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, 0, output.Threshold)
	assert.Equal(t, 25, output.TotalPossible)
	assert.InDelta(t, 0.0, output.Percentage, 0.01)
	assert.Len(t, output.Steps, 2)
}

func TestGetScoringUseCase_NoStepsNoConfig(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewGetScoringUseCase(stepRepo, scoringRepo)

	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, 0, output.Threshold)
	assert.Equal(t, 0, output.TotalPossible)
	assert.InDelta(t, 0.0, output.Percentage, 0.01)
	assert.Empty(t, output.Steps)
}

func TestGetScoringUseCase_ZeroScoreSteps(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewGetScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta sem pontuacao", domain.StepDataTypeFreeText, true, 0)
	stepRepo.addStep(s1)

	config, _ := domain.NewScoringConfig("cfg-1", "spec-1", 1)
	scoringRepo.addScoringConfig(config)

	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Equal(t, 1, output.Threshold)
	assert.Equal(t, 0, output.TotalPossible)
	// total is 0, so percentage stays 0 (no division by zero)
	assert.InDelta(t, 0.0, output.Percentage, 0.01)
}

func TestGetScoringUseCase_StepItemsContent(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewGetScoringUseCase(stepRepo, scoringRepo)

	s, _ := domain.NewStep("step-1", "spec-1", 1, "Qual a area?", domain.StepDataTypeSelection, true, 15)
	stepRepo.addStep(s)

	output, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	require.Len(t, output.Steps, 1)
	assert.Equal(t, "step-1", output.Steps[0].StepID)
	assert.Equal(t, "Qual a area?", output.Steps[0].Text)
	assert.Equal(t, 15, output.Steps[0].Score)
}

// --- UpdateScoringUseCase ---

func TestUpdateScoringUseCase_Success_NewConfig(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 50)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Pergunta 2", domain.StepDataTypeNumber, false, 50)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	output, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    70,
	})

	require.NoError(t, err)
	assert.Equal(t, 70, output.Threshold)
	assert.Equal(t, 100, output.TotalPossible)
	assert.InDelta(t, 70.0, output.Percentage, 0.01)
	assert.Len(t, output.Steps, 2)
}

func TestUpdateScoringUseCase_Success_UpdateExisting(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 100)
	stepRepo.addStep(s1)

	config, _ := domain.NewScoringConfig("cfg-1", "spec-1", 50)
	scoringRepo.addScoringConfig(config)

	output, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    80,
	})

	require.NoError(t, err)
	assert.Equal(t, 80, output.Threshold)
	assert.Equal(t, 100, output.TotalPossible)
	assert.InDelta(t, 80.0, output.Percentage, 0.01)
}

func TestUpdateScoringUseCase_ThresholdExceedsTotal(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 30)
	stepRepo.addStep(s1)

	config, _ := domain.NewScoringConfig("cfg-1", "spec-1", 20)
	scoringRepo.addScoringConfig(config)

	_, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    50, // exceeds total of 30
	})

	assert.ErrorIs(t, err, domain.ErrScoringThresholdExceedsTotal)
}

func TestUpdateScoringUseCase_ZeroThreshold(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 50)
	stepRepo.addStep(s1)

	_, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    0,
	})

	assert.ErrorIs(t, err, domain.ErrScoringThresholdInvalid)
}

func TestUpdateScoringUseCase_NegativeThreshold(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 50)
	stepRepo.addStep(s1)

	_, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    -10,
	})

	assert.ErrorIs(t, err, domain.ErrScoringThresholdInvalid)
}

func TestUpdateScoringUseCase_PersistsConfig(t *testing.T) {
	stepRepo := newMockStepRepo()
	scoringRepo := newMockScoringConfigRepo()
	uc := NewUpdateScoringUseCase(stepRepo, scoringRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Pergunta 1", domain.StepDataTypeFreeText, true, 100)
	stepRepo.addStep(s1)

	_, err := uc.Execute(context.Background(), UpdateScoringInput{
		SpecialistID: "spec-1",
		Threshold:    60,
	})

	require.NoError(t, err)
	stored := scoringRepo.configs["spec-1"]
	require.NotNil(t, stored)
	assert.Equal(t, 60, stored.Threshold)
}
