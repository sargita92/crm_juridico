package domain

import (
	"context"
	"io"
)

// Storage is the port for persisting file bytes. Implementations must scope
// writes and reads under a tenant-aware key to prevent cross-tenant leakage.
type Storage interface {
	Save(ctx context.Context, key string, content []byte) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Size(ctx context.Context, key string) (int64, error)
}
