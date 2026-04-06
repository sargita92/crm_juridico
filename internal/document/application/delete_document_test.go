package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestDeleteDocumentUseCase_Success(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewDeleteDocumentUseCase(docRepo, specDocRepo)

	doc, _ := domain.NewDocument("doc-1", "Doc para excluir", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)

	err := uc.Execute(context.Background(), "doc-1")

	require.NoError(t, err)
	assert.Empty(t, docRepo.documents)
}

func TestDeleteDocumentUseCase_NotFound(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewDeleteDocumentUseCase(docRepo, specDocRepo)

	err := uc.Execute(context.Background(), "nonexistent")

	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestDeleteDocumentUseCase_DissociatesBeforeDelete(t *testing.T) {
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewDeleteDocumentUseCase(docRepo, specDocRepo)

	doc, _ := domain.NewDocument("doc-1", "Doc Compartilhado", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)
	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")
	_ = specDocRepo.Associate(context.Background(), "spec-2", "doc-1")

	err := uc.Execute(context.Background(), "doc-1")

	require.NoError(t, err)
	assert.Empty(t, docRepo.documents)

	ids, _ := specDocRepo.FindSpecialistIDsByDocumentID(context.Background(), "doc-1")
	assert.Empty(t, ids)
}
