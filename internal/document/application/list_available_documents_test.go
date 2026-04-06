package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestListAvailableDocumentsUseCase_FiltersAssociated(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	doc1, _ := domain.NewDocument("doc-1", "Doc Associado", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Doc Disponivel", "txt", "path/2.txt", 512)
	doc3, _ := domain.NewDocument("doc-3", "Doc Disponivel 2", "docx", "path/3.docx", 768)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)
	docRepo.addDocument(doc3)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Len(t, items, 2)
	for _, item := range items {
		assert.NotEqual(t, "doc-1", item.ID)
	}
}

func TestListAvailableDocumentsUseCase_WithSearch(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	doc1, _ := domain.NewDocument("doc-1", "Legislacao Trabalhista", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Contrato Social", "txt", "path/2.txt", 512)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)

	items, err := uc.Execute(context.Background(), "spec-1", "Legislacao")

	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Legislacao Trabalhista", items[0].Name)
}

func TestListAvailableDocumentsUseCase_AllAssociated(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	doc1, _ := domain.NewDocument("doc-1", "Doc 1", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc1)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestListAvailableDocumentsUseCase_NoDocuments(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	items, err := uc.Execute(context.Background(), "spec-1", "")

	require.NoError(t, err)
	assert.Empty(t, items)
}
