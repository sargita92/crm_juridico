package application

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// CrossSellRuleEvaluator evaluates a list of active CrossSellRules against a message and
// optional step-answer map, returning the first matching rule by Ordem ascending, or nil.
type CrossSellRuleEvaluator struct{}

// NewCrossSellRuleEvaluator returns a new CrossSellRuleEvaluator.
func NewCrossSellRuleEvaluator() *CrossSellRuleEvaluator { return &CrossSellRuleEvaluator{} }

// Evaluate returns the first matching active rule (sorted by Ordem ascending), or nil.
func (e *CrossSellRuleEvaluator) Evaluate(
	rules []*specDomain.CrossSellRule,
	message string,
	stepAnswers map[string]string,
) *specDomain.CrossSellRule {
	if len(rules) == 0 {
		return nil
	}
	sorted := make([]*specDomain.CrossSellRule, len(rules))
	copy(sorted, rules)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Ordem < sorted[j].Ordem
	})
	normMsg := normalize(message)
	for _, r := range sorted {
		if !r.Ativo {
			continue
		}
		if matchRule(r, normMsg, stepAnswers) {
			return r
		}
	}
	return nil
}

// matchRule checks whether a single rule matches the normalized message / step answers.
func matchRule(r *specDomain.CrossSellRule, normMsg string, answers map[string]string) bool {
	switch r.TriggerType {
	case specDomain.CrossSellTriggerKeyword:
		kw, ok := r.TriggerConfig.(specDomain.KeywordTrigger)
		if !ok {
			return false
		}
		for _, t := range kw.Termos {
			if strings.Contains(normMsg, normalize(t)) {
				return true
			}
		}
	case specDomain.CrossSellTriggerStepAnswer:
		sa, ok := r.TriggerConfig.(specDomain.StepAnswerTrigger)
		if !ok {
			return false
		}
		ans, found := answers[sa.StepID]
		if !found {
			return false
		}
		re, err := regexp.Compile(sa.Regex)
		if err != nil {
			return false
		}
		return re.MatchString(ans)
	}
	return false
}

// normalize lowercases s and strips diacritical marks so that accent-variant spellings
// compare equal (e.g. "TRABALHISTA" == "trabalhísta" after normalization).
func normalize(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, _ := transform.String(t, strings.ToLower(s))
	return out
}
