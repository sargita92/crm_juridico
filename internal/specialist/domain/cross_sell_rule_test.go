package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestNewCrossSellRule_ValidKeyword(t *testing.T) {
	rule, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"trabalhista"}}, "prod-2")
	require.NoError(t, err)
	assert.Equal(t, domain.CrossSellTriggerKeyword, rule.TriggerType)
	assert.True(t, rule.Ativo)
}

func TestNewCrossSellRule_RejectsEmptyKeywords(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{}}, "prod-2")
	require.ErrorIs(t, err, domain.ErrKeywordTriggerEmpty)
}

func TestNewCrossSellRule_RejectsInvalidStepRegex(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerStepAnswer,
		domain.StepAnswerTrigger{StepID: "step-1", Regex: "(invalid"}, "prod-2")
	require.ErrorIs(t, err, domain.ErrInvalidRegex)
}

func TestNewCrossSellRule_ValidStepAnswer(t *testing.T) {
	rule, err := domain.NewCrossSellRule("id", "spec-1", 2, domain.CrossSellTriggerStepAnswer,
		domain.StepAnswerTrigger{StepID: "step-1", Regex: `^sim$`}, "prod-3")
	require.NoError(t, err)
	assert.Equal(t, domain.CrossSellTriggerStepAnswer, rule.TriggerType)
	assert.True(t, rule.Ativo)
	assert.Equal(t, "prod-3", rule.TargetProductID)
}

func TestNewCrossSellRule_RejectsMissingSpecialistID(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "", 1, domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"termo"}}, "prod-2")
	require.ErrorIs(t, err, domain.ErrSpecialistIDRequired)
}

func TestNewCrossSellRule_RejectsMissingTargetProductID(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"termo"}}, "")
	require.ErrorIs(t, err, domain.ErrTargetProductRequired)
}

func TestNewCrossSellRule_RejectsBlankOnlyKeywordTerm(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerKeyword,
		domain.KeywordTrigger{Termos: []string{"   "}}, "prod-2")
	require.ErrorIs(t, err, domain.ErrKeywordTriggerEmpty)
}

func TestNewCrossSellRule_RejectsUnsupportedTrigger(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerType("unknown"),
		nil, "prod-2")
	require.ErrorIs(t, err, domain.ErrUnsupportedTrigger)
}

func TestNewCrossSellRule_RejectsStepAnswerWithEmptyStepID(t *testing.T) {
	_, err := domain.NewCrossSellRule("id", "spec-1", 1, domain.CrossSellTriggerStepAnswer,
		domain.StepAnswerTrigger{StepID: "", Regex: `.*`}, "prod-2")
	require.ErrorIs(t, err, domain.ErrStepAnswerTriggerInvalid)
}
