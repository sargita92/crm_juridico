package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestUploadDocumentUseCase_Success(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := NewUploadDocumentUseCase(repo)

	output, err := uc.Execute(context.Background(), UploadDocumentInput{
		Name:     "Legislacao Trabalhista",
		Type:     "pdf",
		FilePath: "storage/documents/test.pdf",
		FileSize: 2048,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.ID)
	assert.Equal(t, "Legislacao Trabalhista", output.Name)
	assert.Equal(t, "pdf", output.Type)
	assert.Len(t, repo.documents, 1)
}

func TestUploadDocumentUseCase_EmptyName_ReturnsError(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := NewUploadDocumentUseCase(repo)

	_, err := uc.Execute(context.Background(), UploadDocumentInput{
		Name:     "",
		Type:     "pdf",
		FilePath: "storage/documents/test.pdf",
		FileSize: 1024,
	})

	assert.ErrorIs(t, err, domain.ErrDocumentNameRequired)
}

func TestUploadDocumentUseCase_InvalidType_ReturnsError(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := NewUploadDocumentUseCase(repo)

	_, err := uc.Execute(context.Background(), UploadDocumentInput{
		Name:     "Malware",
		Type:     "exe",
		FilePath: "storage/documents/test.exe",
		FileSize: 1024,
	})

	assert.ErrorIs(t, err, domain.ErrDocumentTypeNotAllowed)
}

func TestUploadDocumentUseCase_EmptyFilePath_ReturnsError(t *testing.T) {
	repo := newMockDocumentRepo()
	uc := NewUploadDocumentUseCase(repo)

	_, err := uc.Execute(context.Background(), UploadDocumentInput{
		Name:     "Doc",
		Type:     "pdf",
		FilePath: "",
		FileSize: 1024,
	})

	assert.ErrorIs(t, err, domain.ErrDocumentFilePathRequired)
}
