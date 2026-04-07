package application

import (
	"testing"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeStep(dataType specDomain.StepDataType) *specDomain.Step {
	return &specDomain.Step{
		ID:       "s1",
		DataType: dataType,
		Score:    10,
	}
}

func TestStepEvaluator_Number(t *testing.T) {
	e := NewStepEvaluator()
	step := makeStep(specDomain.StepDataTypeNumber)

	t.Run("valid integer", func(t *testing.T) {
		res := e.EvaluateByRule(step, "42")
		assert.True(t, res.Completed)
		assert.Equal(t, "42", res.CollectedData)
	})

	t.Run("valid decimal with dot", func(t *testing.T) {
		res := e.EvaluateByRule(step, "3.14")
		assert.True(t, res.Completed)
	})

	t.Run("valid decimal with comma", func(t *testing.T) {
		res := e.EvaluateByRule(step, "1,50")
		assert.True(t, res.Completed)
	})

	t.Run("invalid - letters", func(t *testing.T) {
		res := e.EvaluateByRule(step, "abc")
		assert.False(t, res.Completed)
	})

	t.Run("invalid - empty", func(t *testing.T) {
		res := e.EvaluateByRule(step, "")
		assert.False(t, res.Completed)
	})
}

func TestStepEvaluator_Date(t *testing.T) {
	e := NewStepEvaluator()
	step := makeStep(specDomain.StepDataTypeDate)

	t.Run("valid slash format", func(t *testing.T) {
		res := e.EvaluateByRule(step, "25/12/2024")
		assert.True(t, res.Completed)
	})

	t.Run("valid dash format", func(t *testing.T) {
		res := e.EvaluateByRule(step, "25-12-2024")
		assert.True(t, res.Completed)
	})

	t.Run("invalid", func(t *testing.T) {
		res := e.EvaluateByRule(step, "not a date")
		assert.False(t, res.Completed)
	})
}

func TestStepEvaluator_Document(t *testing.T) {
	e := NewStepEvaluator()
	step := makeStep(specDomain.StepDataTypeDocument)

	t.Run("valid CPF with punctuation", func(t *testing.T) {
		res := e.EvaluateByRule(step, "123.456.789-09")
		assert.True(t, res.Completed)
	})

	t.Run("valid CPF digits only", func(t *testing.T) {
		res := e.EvaluateByRule(step, "12345678909")
		assert.True(t, res.Completed)
	})

	t.Run("valid CNPJ with punctuation", func(t *testing.T) {
		res := e.EvaluateByRule(step, "11.222.333/0001-81")
		assert.True(t, res.Completed)
	})

	t.Run("invalid", func(t *testing.T) {
		res := e.EvaluateByRule(step, "not-a-doc")
		assert.False(t, res.Completed)
	})
}

func TestStepEvaluator_Selection(t *testing.T) {
	e := NewStepEvaluator()
	step := makeStep(specDomain.StepDataTypeSelection)

	t.Run("non-empty message", func(t *testing.T) {
		res := e.EvaluateByRule(step, "opcao A")
		assert.True(t, res.Completed)
		assert.Equal(t, "opcao A", res.CollectedData)
	})

	t.Run("empty message", func(t *testing.T) {
		res := e.EvaluateByRule(step, "")
		assert.False(t, res.Completed)
	})
}

func TestStepEvaluator_FreeText(t *testing.T) {
	e := NewStepEvaluator()
	step := makeStep(specDomain.StepDataTypeFreeText)

	t.Run("always returns false", func(t *testing.T) {
		res := e.EvaluateByRule(step, "qualquer texto longo aqui")
		assert.False(t, res.Completed)
	})
}

func TestStepEvaluator_ParseMetadata(t *testing.T) {
	e := NewStepEvaluator()

	t.Run("valid meta comment", func(t *testing.T) {
		response := `Entendido! Vou registrar sua resposta.<!--STEP_META:{"step_completed":true,"collected_data":"algum dado","score":5}-->`
		meta, cleaned := e.ParseMetadata(response)

		require.True(t, meta.StepCompleted)
		assert.Equal(t, "algum dado", meta.CollectedData)
		assert.Equal(t, 5, meta.Score)
		assert.Equal(t, "Entendido! Vou registrar sua resposta.", cleaned)
	})

	t.Run("no meta comment returns original response", func(t *testing.T) {
		response := "Resposta normal sem meta."
		meta, cleaned := e.ParseMetadata(response)

		assert.False(t, meta.StepCompleted)
		assert.Equal(t, "", meta.CollectedData)
		assert.Equal(t, response, cleaned)
	})
}
