package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestListDocumentsUseCase_Success(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListDocumentsUseCase(docRepo, specDocRepo)

	doc1, _ := domain.NewDocument("doc-1", "Legislacao Trabalhista", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Contrato Social", "docx", "path/2.docx", 2048)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)

	output, err := uc.Execute(context.Background(), ListDocumentsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(2), output.Total)
	assert.Len(t, output.Documents, 2)
}

func TestListDocumentsUseCase_WithSearch(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListDocumentsUseCase(docRepo, specDocRepo)

	doc1, _ := domain.NewDocument("doc-1", "Legislacao Trabalhista", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Contrato Social", "docx", "path/2.docx", 2048)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)

	output, err := uc.Execute(context.Background(), ListDocumentsInput{Search: "Legislacao", Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Equal(t, "Legislacao Trabalhista", output.Documents[0].Name)
}

func TestListDocumentsUseCase_Empty(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListDocumentsUseCase(docRepo, specDocRepo)

	output, err := uc.Execute(context.Background(), ListDocumentsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(0), output.Total)
	assert.Empty(t, output.Documents)
}

func TestListDocumentsUseCase_WithSpecialistCount(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListDocumentsUseCase(docRepo, specDocRepo)

	doc, _ := domain.NewDocument("doc-1", "Doc Compartilhado", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")
	_ = specDocRepo.Associate(context.Background(), "spec-2", "doc-1")

	output, err := uc.Execute(context.Background(), ListDocumentsInput{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, 2, output.Documents[0].SpecialistCount)
}
