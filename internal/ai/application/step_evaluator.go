package application

import (
	"encoding/json"
	"regexp"
	"strings"

	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

var (
	numberRegex   = regexp.MustCompile(`^\d+([.,]\d+)?$`)
	dateRegex     = regexp.MustCompile(`\d{2}[/\-]\d{2}[/\-]\d{4}`)
	cpfRegex      = regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`)
	cnpjRegex     = regexp.MustCompile(`\d{2}\.?\d{3}\.?\d{3}/?\d{4}-?\d{2}`)
	stepMetaRegex = regexp.MustCompile(`<!--STEP_META:(.*?)-->`)
)

// StepEvalResult holds the outcome of evaluating a step against a user message.
type StepEvalResult struct {
	Completed     bool
	CollectedData string
}

// StepMetadata holds structured metadata returned by the LLM inside a hidden comment.
type StepMetadata struct {
	StepCompleted bool   `json:"step_completed"`
	CollectedData string `json:"collected_data"`
	Score         int    `json:"score"`
}

// StepEvaluator evaluates user messages against step rules.
type StepEvaluator struct{}

// NewStepEvaluator creates a new StepEvaluator.
func NewStepEvaluator() *StepEvaluator {
	return &StepEvaluator{}
}

// EvaluateByRule checks whether a message satisfies the step's data type rule.
// free_text steps always return false (requires LLM evaluation via ParseMetadata).
func (e *StepEvaluator) EvaluateByRule(step *specDomain.Step, message string) StepEvalResult {
	msg := strings.TrimSpace(message)
	switch step.DataType {
	case specDomain.StepDataTypeNumber:
		if numberRegex.MatchString(msg) {
			return StepEvalResult{Completed: true, CollectedData: msg}
		}
	case specDomain.StepDataTypeDate:
		if dateRegex.MatchString(msg) {
			return StepEvalResult{Completed: true, CollectedData: msg}
		}
	case specDomain.StepDataTypeDocument:
		if cpfRegex.MatchString(msg) || cnpjRegex.MatchString(msg) {
			return StepEvalResult{Completed: true, CollectedData: msg}
		}
	case specDomain.StepDataTypeSelection:
		if msg != "" {
			return StepEvalResult{Completed: true, CollectedData: msg}
		}
	case specDomain.StepDataTypeFreeText:
		// Requires LLM evaluation; always returns false here.
		return StepEvalResult{Completed: false}
	}
	return StepEvalResult{Completed: false}
}

// ParseMetadata extracts a <!--STEP_META:JSON--> comment from an LLM response,
// parses the JSON into StepMetadata, and returns both the metadata and the
// cleaned response (with the comment removed).
func (e *StepEvaluator) ParseMetadata(response string) (StepMetadata, string) {
	match := stepMetaRegex.FindStringSubmatchIndex(response)
	if match == nil {
		return StepMetadata{}, response
	}

	jsonStr := response[match[2]:match[3]]
	cleaned := response[:match[0]] + response[match[1]:]
	cleaned = strings.TrimSpace(cleaned)

	var meta StepMetadata
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		return StepMetadata{}, response
	}

	return meta, cleaned
}
