package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- CreateStepUseCase ---

func TestCreateStepUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	output, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "spec-1",
		Text:         "Qual o seu nome?",
		DataType:     "free_text",
		Required:     true,
		Score:        10,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "spec-1", output.SpecialistID)
	assert.Equal(t, 1, output.OrderIndex)
	assert.Equal(t, "Qual o seu nome?", output.Text)
	assert.Equal(t, "free_text", output.DataType)
	assert.True(t, output.Required)
	assert.Equal(t, 10, output.Score)
}

func TestCreateStepUseCase_OrderIncrementsCorrectly(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	existing, _ := domain.NewStep("step-existing", "spec-1", 1, "Primeiro passo", domain.StepDataTypeFreeText, false, 5)
	stepRepo.addStep(existing)

	output, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "spec-1",
		Text:         "Segundo passo",
		DataType:     "number",
		Required:     false,
		Score:        5,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, output.OrderIndex)
}

func TestCreateStepUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	_, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "nonexistent",
		Text:         "Pergunta",
		DataType:     "free_text",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistNotFound)
}

func TestCreateStepUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	_ = spec.Deactivate()
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "spec-1",
		Text:         "Pergunta",
		DataType:     "free_text",
	})

	assert.ErrorIs(t, err, domain.ErrSpecialistInactive)
}

func TestCreateStepUseCase_InvalidDataType(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "spec-1",
		Text:         "Pergunta",
		DataType:     "invalid_type",
	})

	assert.ErrorIs(t, err, domain.ErrStepDataTypeInvalid)
}

func TestCreateStepUseCase_NegativeScore(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	stepRepo := newMockStepRepo()
	uc := NewCreateStepUseCase(specRepo, stepRepo)

	spec, _ := domain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	_, err := uc.Execute(context.Background(), CreateStepInput{
		SpecialistID: "spec-1",
		Text:         "Pergunta",
		DataType:     "free_text",
		Score:        -1,
	})

	assert.ErrorIs(t, err, domain.ErrStepScoreNegative)
}

// --- UpdateStepUseCase ---

func TestUpdateStepUseCase_Success(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewUpdateStepUseCase(stepRepo)

	s, _ := domain.NewStep("step-1", "spec-1", 1, "Texto antigo", domain.StepDataTypeFreeText, false, 5)
	stepRepo.addStep(s)

	output, err := uc.Execute(context.Background(), UpdateStepInput{
		ID:       "step-1",
		Text:     "Texto novo",
		DataType: "number",
		Required: true,
		Score:    20,
	})

	require.NoError(t, err)
	assert.Equal(t, "step-1", output.ID)
	assert.Equal(t, "Texto novo", output.Text)
	assert.Equal(t, "number", output.DataType)
	assert.True(t, output.Required)
	assert.Equal(t, 20, output.Score)
}

func TestUpdateStepUseCase_NotFound(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewUpdateStepUseCase(stepRepo)

	_, err := uc.Execute(context.Background(), UpdateStepInput{
		ID:       "nonexistent",
		Text:     "Texto",
		DataType: "free_text",
	})

	assert.ErrorIs(t, err, domain.ErrStepNotFound)
}

func TestUpdateStepUseCase_EmptyText(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewUpdateStepUseCase(stepRepo)

	s, _ := domain.NewStep("step-1", "spec-1", 1, "Texto original", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s)

	_, err := uc.Execute(context.Background(), UpdateStepInput{
		ID:       "step-1",
		Text:     "",
		DataType: "free_text",
	})

	assert.ErrorIs(t, err, domain.ErrStepTextRequired)
}

// --- DeleteStepUseCase ---

func TestDeleteStepUseCase_Success(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewDeleteStepUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeFreeText, false, 0)
	s3, _ := domain.NewStep("step-3", "spec-1", 3, "Passo 3", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)
	stepRepo.addStep(s3)

	err := uc.Execute(context.Background(), "step-1")

	require.NoError(t, err)
	_, findErr := stepRepo.FindByID(context.Background(), "step-1")
	assert.ErrorIs(t, findErr, domain.ErrStepNotFound)
	// After delete of order 1, step-2 (order 2) becomes 1, step-3 (order 3) becomes 2
	assert.Equal(t, 1, stepRepo.steps["step-2"].OrderIndex)
	assert.Equal(t, 2, stepRepo.steps["step-3"].OrderIndex)
}

func TestDeleteStepUseCase_NotFound(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewDeleteStepUseCase(stepRepo)

	err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrStepNotFound)
}

func TestDeleteStepUseCase_SingleStep(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewDeleteStepUseCase(stepRepo)

	s, _ := domain.NewStep("step-1", "spec-1", 1, "Unico passo", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s)

	err := uc.Execute(context.Background(), "step-1")

	require.NoError(t, err)
	assert.Empty(t, stepRepo.steps)
}

// --- MoveStepUseCase ---

func TestMoveStepUseCase_MoveDown(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "step-1", Direction: "down"})

	require.NoError(t, err)
	assert.Equal(t, 2, stepRepo.steps["step-1"].OrderIndex)
	assert.Equal(t, 1, stepRepo.steps["step-2"].OrderIndex)
}

func TestMoveStepUseCase_MoveUp(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "step-2", Direction: "up"})

	require.NoError(t, err)
	assert.Equal(t, 2, stepRepo.steps["step-1"].OrderIndex)
	assert.Equal(t, 1, stepRepo.steps["step-2"].OrderIndex)
}

func TestMoveStepUseCase_AlreadyFirst(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "step-1", Direction: "up"})

	assert.ErrorIs(t, err, ErrStepAlreadyFirst)
}

func TestMoveStepUseCase_AlreadyLast(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "step-2", Direction: "down"})

	assert.ErrorIs(t, err, ErrStepAlreadyLast)
}

func TestMoveStepUseCase_InvalidDirection(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "step-1", Direction: "sideways"})

	assert.ErrorIs(t, err, ErrMoveDirectionInvalid)
}

func TestMoveStepUseCase_StepNotFound(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewMoveStepUseCase(stepRepo)

	err := uc.Execute(context.Background(), MoveStepInput{ID: "nonexistent", Direction: "up"})

	assert.ErrorIs(t, err, domain.ErrStepNotFound)
}

// --- ListStepsUseCase ---

func TestListStepsUseCase_Success(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewListStepsUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo 1", domain.StepDataTypeFreeText, true, 10)
	s2, _ := domain.NewStep("step-2", "spec-1", 2, "Passo 2", domain.StepDataTypeNumber, false, 5)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 2)
	// Results are ordered by OrderIndex
	assert.Equal(t, "step-1", items[0].ID)
	assert.Equal(t, 1, items[0].OrderIndex)
	assert.Equal(t, "step-2", items[1].ID)
	assert.Equal(t, 2, items[1].OrderIndex)
}

func TestListStepsUseCase_Empty(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewListStepsUseCase(stepRepo)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListStepsUseCase_FiltersBySpecialist(t *testing.T) {
	stepRepo := newMockStepRepo()
	uc := NewListStepsUseCase(stepRepo)

	s1, _ := domain.NewStep("step-1", "spec-1", 1, "Passo de spec-1", domain.StepDataTypeFreeText, false, 0)
	s2, _ := domain.NewStep("step-2", "spec-2", 1, "Passo de spec-2", domain.StepDataTypeFreeText, false, 0)
	stepRepo.addStep(s1)
	stepRepo.addStep(s2)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "spec-1", items[0].SpecialistID)
}
