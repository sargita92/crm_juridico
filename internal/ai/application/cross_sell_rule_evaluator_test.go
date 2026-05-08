package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/ai/application"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// mustNewKeywordRule creates a keyword-based CrossSellRule for tests, panics on error.
func mustNewKeywordRule(t *testing.T, specialistID string, ordem int, termos []string, productID string) *specDomain.CrossSellRule {
	t.Helper()
	r, err := specDomain.NewCrossSellRule(
		specialistID+"-rule",
		specialistID,
		ordem,
		specDomain.CrossSellTriggerKeyword,
		specDomain.KeywordTrigger{Termos: termos},
		productID,
	)
	require.NoError(t, err)
	return r
}

// --- Core tests from plan ---

func TestEvaluator_KeywordMatchCaseInsensitiveAccentsNormalized(t *testing.T) {
	rule := mustNewKeywordRule(t, "spec-1", 0, []string{"trabalhista"}, "prod-2")
	eval := application.NewCrossSellRuleEvaluator()
	match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "Tenho dúvida TRABALHISTA hoje", nil)
	require.NotNil(t, match)
	assert.Equal(t, rule.ID, match.ID)
}

func TestEvaluator_FirstByOrdemWins(t *testing.T) {
	r1 := mustNewKeywordRule(t, "spec-1", 0, []string{"trabalhista"}, "prod-A")
	r2 := mustNewKeywordRule(t, "spec-1", 1, []string{"trabalhista"}, "prod-B")
	eval := application.NewCrossSellRuleEvaluator()
	match := eval.Evaluate([]*specDomain.CrossSellRule{r1, r2}, "trabalhista", nil)
	assert.Equal(t, r1.ID, match.ID)
}

func TestEvaluator_InactiveRuleIsSkipped(t *testing.T) {
	r := mustNewKeywordRule(t, "spec-1", 0, []string{"x"}, "prod-A")
	r.Ativo = false
	eval := application.NewCrossSellRuleEvaluator()
	assert.Nil(t, eval.Evaluate([]*specDomain.CrossSellRule{r}, "x", nil))
}

func TestEvaluator_StepAnswerRegexMatch(t *testing.T) {
	rule, err := specDomain.NewCrossSellRule("id", "spec-1", 0, specDomain.CrossSellTriggerStepAnswer,
		specDomain.StepAnswerTrigger{StepID: "step-2", Regex: `(?i)^sim$`}, "prod-2")
	require.NoError(t, err)
	eval := application.NewCrossSellRuleEvaluator()
	match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "irrelevante", map[string]string{"step-2": "Sim"})
	require.NotNil(t, match)
}

// --- Additional edge cases ---

func TestEvaluator_EmptyRulesListReturnsNil(t *testing.T) {
	eval := application.NewCrossSellRuleEvaluator()
	assert.Nil(t, eval.Evaluate(nil, "qualquer coisa", nil))
}

func TestEvaluator_NoMatchReturnsNil(t *testing.T) {
	r := mustNewKeywordRule(t, "spec-1", 0, []string{"imobiliario"}, "prod-A")
	eval := application.NewCrossSellRuleEvaluator()
	assert.Nil(t, eval.Evaluate([]*specDomain.CrossSellRule{r}, "assunto trabalhista", nil))
}

func TestEvaluator_StepAnswerMissingAnswerReturnsNil(t *testing.T) {
	rule, err := specDomain.NewCrossSellRule("id", "spec-1", 0, specDomain.CrossSellTriggerStepAnswer,
		specDomain.StepAnswerTrigger{StepID: "step-2", Regex: `(?i)^sim$`}, "prod-2")
	require.NoError(t, err)
	eval := application.NewCrossSellRuleEvaluator()
	// stepAnswers map does not contain step-2
	match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "sim", map[string]string{"step-1": "sim"})
	assert.Nil(t, match)
}

func TestEvaluator_AccentNormalizationKeyword(t *testing.T) {
	// term stored without accent, message has accent (and vice versa)
	rule := mustNewKeywordRule(t, "spec-1", 0, []string{"trabalhista"}, "prod-A")
	eval := application.NewCrossSellRuleEvaluator()
	// message has accent on 'i' — should still match after normalization
	match := eval.Evaluate([]*specDomain.CrossSellRule{rule}, "trabalhísta", nil)
	require.NotNil(t, match)
	assert.Equal(t, rule.ID, match.ID)
}

func TestEvaluator_OrdemUnorderedInputSorted(t *testing.T) {
	// r2 has lower Ordem — must win even when passed second
	r1 := mustNewKeywordRule(t, "spec-1", 10, []string{"trabalhista"}, "prod-A")
	r2 := mustNewKeywordRule(t, "spec-2", 5, []string{"trabalhista"}, "prod-B")
	eval := application.NewCrossSellRuleEvaluator()
	match := eval.Evaluate([]*specDomain.CrossSellRule{r1, r2}, "trabalhista", nil)
	assert.Equal(t, r2.ID, match.ID)
}
