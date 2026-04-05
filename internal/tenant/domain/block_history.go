package domain

import (
	"context"
	"errors"
	"time"
)

type BlockAction string

const (
	BlockActionBlock   BlockAction = "block"
	BlockActionUnblock BlockAction = "unblock"
)

var (
	ErrHistoryTenantIDRequired    = errors.New("tenant ID is required for history entry")
	ErrHistoryPerformedByRequired = errors.New("performed by is required for history entry")
	ErrInvalidBlockAction         = errors.New("invalid block action: must be block or unblock")
)

type BlockHistoryEntry struct {
	ID          string
	TenantID    string
	Action      BlockAction
	Reason      string
	PerformedBy string
	CreatedAt   time.Time
}

func NewBlockHistoryEntry(id, tenantID string, action BlockAction, reason, performedBy string) (*BlockHistoryEntry, error) {
	if tenantID == "" {
		return nil, ErrHistoryTenantIDRequired
	}
	if reason == "" {
		return nil, ErrBlockReasonRequired
	}
	if len(reason) > 500 {
		return nil, ErrReasonTooLong
	}
	if performedBy == "" {
		return nil, ErrHistoryPerformedByRequired
	}
	if action != BlockActionBlock && action != BlockActionUnblock {
		return nil, ErrInvalidBlockAction
	}

	return &BlockHistoryEntry{
		ID:          id,
		TenantID:    tenantID,
		Action:      action,
		Reason:      reason,
		PerformedBy: performedBy,
		CreatedAt:   time.Now(),
	}, nil
}

type BlockHistoryRepository interface {
	Save(ctx context.Context, entry *BlockHistoryEntry) error
	FindByTenantID(ctx context.Context, tenantID string) ([]BlockHistoryEntry, error)
}
