package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrToolAlreadyAssociated = errors.New("tool is already associated with this specialist")
	ErrToolNotAssociated     = errors.New("tool is not associated with this specialist")
)

type SpecialistTool struct {
	ID           string
	SpecialistID string
	ToolName     string
	CreatedAt    time.Time
}

type SpecialistToolRepository interface {
	Associate(ctx context.Context, specialistID, toolName string) error
	Dissociate(ctx context.Context, specialistID, toolName string) error
	FindToolNamesBySpecialistID(ctx context.Context, specialistID string) ([]string, error)
	FindAll(ctx context.Context, specialistID string) ([]SpecialistTool, error)
}
