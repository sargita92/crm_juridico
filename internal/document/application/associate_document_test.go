package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestAssociateDocumentUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	doc, _ := domain.NewDocument("doc-1", "Doc", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  []string{"doc-1"},
	})

	require.NoError(t, err)
	exists, _ := specDocRepo.Exists(context.Background(), "spec-1", "doc-1")
	assert.True(t, exists)
}

func TestAssociateDocumentUseCase_MultipleDocuments(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	doc1, _ := domain.NewDocument("doc-1", "Doc 1", "pdf", "path/1.pdf", 1024)
	doc2, _ := domain.NewDocument("doc-2", "Doc 2", "txt", "path/2.txt", 512)
	docRepo.addDocument(doc1)
	docRepo.addDocument(doc2)

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  []string{"doc-1", "doc-2"},
	})

	require.NoError(t, err)
	ids, _ := specDocRepo.FindDocumentIDsBySpecialistID(context.Background(), "spec-1")
	assert.Len(t, ids, 2)
}

func TestAssociateDocumentUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "nonexistent",
		DocumentIDs:  []string{"doc-1"},
	})

	assert.ErrorIs(t, err, specialistdomain.ErrSpecialistNotFound)
}

func TestAssociateDocumentUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	_ = spec.Deactivate()
	specRepo.addSpecialist(spec)

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  []string{"doc-1"},
	})

	assert.ErrorIs(t, err, specialistdomain.ErrSpecialistInactive)
}

func TestAssociateDocumentUseCase_DocumentNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  []string{"nonexistent"},
	})

	assert.ErrorIs(t, err, domain.ErrDocumentNotFound)
}

func TestAssociateDocumentUseCase_SkipAlreadyAssociated(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	doc, _ := domain.NewDocument("doc-1", "Doc", "pdf", "path/1.pdf", 1024)
	docRepo.addDocument(doc)

	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  []string{"doc-1"},
	})

	require.NoError(t, err)
}

func TestAssociateDocumentUseCase_BatchTooLarge(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	docRepo := newMockDocumentRepo()
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewAssociateDocumentUseCase(specRepo, docRepo, specDocRepo)

	ids := make([]string, MaxAssociationBatchSize+1)
	for i := range ids {
		ids[i] = "doc-" + string(rune('0'+i))
	}

	err := uc.Execute(context.Background(), AssociateDocumentInput{
		SpecialistID: "spec-1",
		DocumentIDs:  ids,
	})

	assert.ErrorIs(t, err, ErrAssociationBatchTooLarge)
}
