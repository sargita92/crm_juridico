package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocument_WithValidData_ReturnsDocument(t *testing.T) {
	d, err := NewDocument("uuid-1", "Legislacao Trabalhista", "pdf", "storage/documents/uuid-1.pdf", 1024)

	require.NoError(t, err)
	assert.Equal(t, "uuid-1", d.ID)
	assert.Equal(t, "Legislacao Trabalhista", d.Name)
	assert.Equal(t, "pdf", d.Type)
	assert.Equal(t, "storage/documents/uuid-1.pdf", d.FilePath)
	assert.Equal(t, int64(1024), d.FileSize)
	assert.False(t, d.CreatedAt.IsZero())
	assert.False(t, d.UpdatedAt.IsZero())
}

func TestNewDocument_EmptyName_ReturnsError(t *testing.T) {
	_, err := NewDocument("uuid-1", "", "pdf", "path/file.pdf", 1024)

	assert.ErrorIs(t, err, ErrDocumentNameRequired)
}

func TestNewDocument_NameTooLong_ReturnsError(t *testing.T) {
	longName := strings.Repeat("a", MaxDocumentNameLength+1)

	_, err := NewDocument("uuid-1", longName, "pdf", "path/file.pdf", 1024)

	assert.ErrorIs(t, err, ErrDocumentNameTooLong)
}

func TestNewDocument_NameAtMaxLength_ReturnsDocument(t *testing.T) {
	name := strings.Repeat("a", MaxDocumentNameLength)

	d, err := NewDocument("uuid-1", name, "pdf", "path/file.pdf", 1024)

	require.NoError(t, err)
	assert.Equal(t, name, d.Name)
}

func TestNewDocument_TypeNotAllowed_ReturnsError(t *testing.T) {
	_, err := NewDocument("uuid-1", "Doc", "exe", "path/file.exe", 1024)

	assert.ErrorIs(t, err, ErrDocumentTypeNotAllowed)
}

func TestNewDocument_AllAllowedTypes(t *testing.T) {
	for _, docType := range AllowedDocumentTypes {
		t.Run(docType, func(t *testing.T) {
			d, err := NewDocument("uuid-1", "Doc", docType, "path/file."+docType, 1024)

			require.NoError(t, err)
			assert.Equal(t, docType, d.Type)
		})
	}
}

func TestNewDocument_EmptyFilePath_ReturnsError(t *testing.T) {
	_, err := NewDocument("uuid-1", "Doc", "pdf", "", 1024)

	assert.ErrorIs(t, err, ErrDocumentFilePathRequired)
}

func TestNewDocument_ZeroFileSize_ReturnsDocument(t *testing.T) {
	d, err := NewDocument("uuid-1", "Doc", "pdf", "path/file.pdf", 0)

	require.NoError(t, err)
	assert.Equal(t, int64(0), d.FileSize)
}

func TestIsAllowedType_ValidTypes(t *testing.T) {
	assert.True(t, IsAllowedType("pdf"))
	assert.True(t, IsAllowedType("txt"))
	assert.True(t, IsAllowedType("docx"))
}

func TestIsAllowedType_InvalidTypes(t *testing.T) {
	assert.False(t, IsAllowedType("exe"))
	assert.False(t, IsAllowedType(""))
	assert.False(t, IsAllowedType("jpg"))
	assert.False(t, IsAllowedType("PDF"))
}
