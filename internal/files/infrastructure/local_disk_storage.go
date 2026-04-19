package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

// LocalDiskStorage persists file bytes under a root directory, scoped by the
// tenant-aware key. The key must NOT contain path traversal segments — the
// Save/Open/Size methods refuse keys that would resolve outside the root.
type LocalDiskStorage struct {
	root string
}

func NewLocalDiskStorage(root string) (*LocalDiskStorage, error) {
	if root == "" {
		return nil, errors.New("storage root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &LocalDiskStorage{root: abs}, nil
}

func (s *LocalDiskStorage) resolve(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", errors.New("storage key is empty")
	}
	// Guard against absolute paths and traversal. Keys must be tenant-scoped
	// relative paths; we join with the root and ensure the result stays inside.
	clean := filepath.Clean("/" + key)
	full := filepath.Join(s.root, clean)
	rel, err := filepath.Rel(s.root, full)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid storage key: %s", key)
	}
	return full, nil
}

func (s *LocalDiskStorage) Save(ctx context.Context, key string, content []byte) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	// Write with restrictive permissions. Use a temp file + rename to avoid
	// partial writes being visible to readers.
	tmp, err := os.CreateTemp(filepath.Dir(full), ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, full); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func (s *LocalDiskStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	full, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, domain.ErrFileContentUnavailable
		}
		return nil, fmt.Errorf("open storage file: %w", err)
	}
	return f, nil
}

func (s *LocalDiskStorage) Size(ctx context.Context, key string) (int64, error) {
	full, err := s.resolve(key)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, domain.ErrFileContentUnavailable
		}
		return 0, err
	}
	return info.Size(), nil
}
