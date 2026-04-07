package domain

import "time"

type StepDataType string

const (
	StepDataTypeFreeText  StepDataType = "free_text"
	StepDataTypeNumber    StepDataType = "number"
	StepDataTypeDate      StepDataType = "date"
	StepDataTypeDocument  StepDataType = "document"
	StepDataTypeSelection StepDataType = "selection"
)

const MaxStepTextLength = 1000

type Step struct {
	ID             string
	SpecialistID   string
	OrderIndex     int
	Text           string
	DataType       StepDataType
	Required       bool
	Score          int
	TargetColumnID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewStep(id, specialistID string, orderIndex int, text string, dataType StepDataType, required bool, score int, targetColumnID string) (*Step, error) {
	if specialistID == "" {
		return nil, ErrSpecialistIDRequired
	}
	if text == "" {
		return nil, ErrStepTextRequired
	}
	if len(text) > MaxStepTextLength {
		return nil, ErrStepTextTooLong
	}
	if !isValidStepDataType(dataType) {
		return nil, ErrStepDataTypeInvalid
	}
	if score < 0 {
		return nil, ErrStepScoreNegative
	}

	now := time.Now()
	return &Step{
		ID: id, SpecialistID: specialistID, OrderIndex: orderIndex,
		Text: text, DataType: dataType, Required: required, Score: score,
		TargetColumnID: targetColumnID,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *Step) Update(text string, dataType StepDataType, required bool, score int, targetColumnID string) error {
	if text == "" {
		return ErrStepTextRequired
	}
	if len(text) > MaxStepTextLength {
		return ErrStepTextTooLong
	}
	if !isValidStepDataType(dataType) {
		return ErrStepDataTypeInvalid
	}
	if score < 0 {
		return ErrStepScoreNegative
	}
	s.Text = text
	s.DataType = dataType
	s.Required = required
	s.Score = score
	s.TargetColumnID = targetColumnID
	s.UpdatedAt = time.Now()
	return nil
}

func isValidStepDataType(dt StepDataType) bool {
	switch dt {
	case StepDataTypeFreeText, StepDataTypeNumber, StepDataTypeDate, StepDataTypeDocument, StepDataTypeSelection:
		return true
	}
	return false
}
