package domain

import "time"

type ColumnType string

const (
	ColumnTypeEntry        ColumnType = "entry"
	ColumnTypeIntermediate ColumnType = "intermediate"
	ColumnTypeWon          ColumnType = "won"
	ColumnTypeLost         ColumnType = "lost"
)

const MaxColumnNameLength = 100
const MaxColumnsPerFunnel = 15

type Column struct {
	ID         string
	FunnelID   string
	Name       string
	OrderIndex int
	Type       ColumnType
	Color      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func NewColumn(id, funnelID, name string, orderIndex int, colType ColumnType, color string) (*Column, error) {
	if funnelID == "" {
		return nil, ErrFunnelIDRequired
	}
	if name == "" {
		return nil, ErrColumnNameRequired
	}
	if len(name) > MaxColumnNameLength {
		return nil, ErrColumnNameTooLong
	}
	if !isValidColumnType(colType) {
		return nil, ErrColumnTypeInvalid
	}
	if color == "" {
		color = "#3b82f6"
	}
	now := time.Now()
	return &Column{
		ID: id, FunnelID: funnelID, Name: name,
		OrderIndex: orderIndex, Type: colType, Color: color,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (c *Column) Update(name string, colType ColumnType, color string) error {
	if name == "" {
		return ErrColumnNameRequired
	}
	if len(name) > MaxColumnNameLength {
		return ErrColumnNameTooLong
	}
	if !isValidColumnType(colType) {
		return ErrColumnTypeInvalid
	}
	c.Name = name
	c.Type = colType
	if color != "" {
		c.Color = color
	}
	c.UpdatedAt = time.Now()
	return nil
}

func isValidColumnType(t ColumnType) bool {
	switch t {
	case ColumnTypeEntry, ColumnTypeIntermediate, ColumnTypeWon, ColumnTypeLost:
		return true
	}
	return false
}
