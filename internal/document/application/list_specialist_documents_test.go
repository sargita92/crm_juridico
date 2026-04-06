package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestListSpecialistDocumentsUseCase_Success(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListSpecialistDocumentsUseCase(specDocRepo, docRepo)

	doc1, _ := domain.NewDocument("doc-1", "Doc 1", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Doc 2", "txt", "path/2.txt", 512)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-2")

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestListSpecialistDocumentsUseCase_Empty(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListSpecialistDocumentsUseCase(specDocRepo, docRepo)

	items, err := uc.Execute(context.Background(), "spec-1")

	require.NoError(t, err)
	assert.Empty(t, items)
}
