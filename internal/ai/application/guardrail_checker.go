package application

import (
	"strings"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// GuardrailChecker checks AI responses against configured guardrails.
type GuardrailChecker struct{}

// NewGuardrailChecker creates a new GuardrailChecker.
func NewGuardrailChecker() *GuardrailChecker {
	return &GuardrailChecker{}
}

// Check inspects the response against the list of guardrails.
// Only active forbidden_topics guardrails are evaluated.
// Returns violated=true and the guardrail's fallback Message if a keyword is found.
func (c *GuardrailChecker) Check(response string, guardrails []specDomain.Guardrail) (violated bool, fallback string) {
	lowerResponse := strings.ToLower(response)

	for _, g := range guardrails {
		if !g.Active {
			continue
		}
		if g.Type != specDomain.GuardrailTypeForbiddenTopics {
			continue
		}

		keywords := strings.Split(g.Rule, ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(strings.ToLower(kw))
			if kw == "" {
				continue
			}
			if strings.Contains(lowerResponse, kw) {
				return true, g.Message
			}
		}
	}

	return false, ""
}
