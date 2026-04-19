package application

import "github.com/google/uuid"

type IDGenerator interface {
	NewID() string
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewID() string { return uuid.NewString() }
