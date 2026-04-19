package domain

import (
	"context"
	"time"
)

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

type FileRepository interface {
	Create(ctx context.Context, f *File) error
	FindByID(ctx context.Context, tenantID, id string) (*File, error)
	List(ctx context.Context, q ListQuery) (*ListResult, error)
	CountByLead(ctx context.Context, tenantID, leadID string) (int64, error)
	ListRecentByLead(ctx context.Context, tenantID, leadID string, limit int) ([]File, error)
}

type ListQuery struct {
	TenantID  string
	LeadID    *string
	MediaType *MediaType
	Search    string
	From      *time.Time
	To        *time.Time
	Page      int
	PageSize  int
}

type ListResult struct {
	Items    []FileWithContext
	Total    int64
	Page     int
	PageSize int
}

type FileWithContext struct {
	File
	ContactName string
	LeadLabel   string
}
