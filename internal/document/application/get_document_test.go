package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestGetDocumentUseCase_Success(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewGetDocumentUseCase(docRepo, specDocRepo)

	doc, _ := domain.NewDocument("doc-1", "Legislacao Trabalhista", "pdf", "path/1.pdf", 2048)
	docRepo.addDocument(doc)

	output, err := uc.Execute(context.Background(), "doc-1")

	require.NoError(t, err)
	assert.Equal(t, "doc-1", output.ID)
	assert.Equal(t, "Legislacao Trabalhista", output.Name)
	assert.Equal(t, "pdf", output.Type)
	assert.Equal(t, int64(2048), output.FileSize)
}

func TestGetDocumentUseCase_NotFound(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewGetDocumentUseCase(docRepo, specDocRepo)

	_, err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestGetDocumentUseCase_WithSpecialistCount(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewGetDocumentUseCase(docRepo, specDocRepo)

	doc, _ := domain.NewDocument("doc-1", "Doc", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")
	_ = specDocRepo.Associate(context.Background(), "spec-2", "doc-1")

	output, err := uc.Execute(context.Background(), "doc-1")

	require.NoError(t, err)
	assert.Equal(t, 2, output.SpecialistCount)
}
