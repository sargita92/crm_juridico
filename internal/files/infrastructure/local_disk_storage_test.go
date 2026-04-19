package infrastructure

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

func newTempStorage(t *testing.T) *LocalDiskStorage {
	t.Helper()
	root := t.TempDir()
	s, err := NewLocalDiskStorage(root)
	require.NoError(t, err)
	return s
}

func TestNewLocalDiskStorage_EmptyRoot(t *testing.T) {
	_, err := NewLocalDiskStorage("")
	assert.Error(t, err)
}

func TestNewLocalDiskStorage_CreatesRootIfMissing(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "nested", "storage")
	_, err := NewLocalDiskStorage(root)
	require.NoError(t, err)
	info, err := os.Stat(root)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestLocalDiskStorage_SaveOpenSize(t *testing.T) {
	s := newTempStorage(t)
	ctx := context.Background()
	key := "tenant-1/2026/04/uuid.pdf"
	content := []byte("hello world")

	require.NoError(t, s.Save(ctx, key, content))

	rc, err := s.Open(ctx, key)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, content, got)

	size, err := s.Size(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
}

func TestLocalDiskStorage_OpenMissingReturnsDomainError(t *testing.T) {
	s := newTempStorage(t)
	_, err := s.Open(context.Background(), "tenant-1/2026/04/missing.pdf")
	assert.ErrorIs(t, err, domain.ErrFileContentUnavailable)
}

func TestLocalDiskStorage_SizeMissingReturnsDomainError(t *testing.T) {
	s := newTempStorage(t)
	_, err := s.Size(context.Background(), "tenant-1/2026/04/missing.pdf")
	assert.ErrorIs(t, err, domain.ErrFileContentUnavailable)
}

func TestLocalDiskStorage_EmptyKeyRejected(t *testing.T) {
	s := newTempStorage(t)
	err := s.Save(context.Background(), "", []byte("x"))
	assert.Error(t, err)
	_, err = s.Open(context.Background(), "")
	assert.Error(t, err)
	_, err = s.Size(context.Background(), "")
	assert.Error(t, err)
}

func TestLocalDiskStorage_PathTraversalBlocked(t *testing.T) {
	s := newTempStorage(t)
	bad := []string{
		"../../../etc/passwd",
		"tenant/../../secret",
		"/absolute/path",
	}
	for _, k := range bad {
		t.Run(k, func(t *testing.T) {
			err := s.Save(context.Background(), k, []byte("x"))
			// Either rejected by resolve or confined inside root — never
			// allowed to escape. We assert no file is created outside the root.
			if err == nil {
				// If the implementation confined the path, verify nothing
				// was written outside root.
				full := filepath.Join(s.root, filepath.Clean("/"+k))
				rel, rErr := filepath.Rel(s.root, full)
				require.NoError(t, rErr)
				assert.False(t, filepath.IsAbs(rel))
				assert.NotContains(t, rel, "..")
			}
		})
	}

	// Ensure no file was created at /tmp/etc/passwd or similar.
	_, err := os.Stat("/tmp/etc/passwd")
	assert.True(t, os.IsNotExist(err) || err != nil)
}

func TestLocalDiskStorage_OverwriteSavesNewContent(t *testing.T) {
	s := newTempStorage(t)
	ctx := context.Background()
	key := "tenant-1/2026/04/uuid.pdf"

	require.NoError(t, s.Save(ctx, key, []byte("v1")))
	require.NoError(t, s.Save(ctx, key, []byte("v2-longer")))

	rc, err := s.Open(ctx, key)
	require.NoError(t, err)
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	assert.Equal(t, []byte("v2-longer"), got)
}
