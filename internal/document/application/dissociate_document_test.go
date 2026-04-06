package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/document/domain"
)

func TestDissociateDocumentUseCase_Success(t *testing.T) {
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewDissociateDocumentUseCase(specDocRepo)

	_ = specDocRepo.Associate(context.Background(), "spec-1", "doc-1")

	err := uc.Execute(context.Background(), "spec-1", "doc-1")

	require.NoError(t, err)
	exists, _ := specDocRepo.Exists(context.Background(), "spec-1", "doc-1")
	assert.False(t, exists)
}

func TestDissociateDocumentUseCase_NotAssociated(t *testing.T) {
	specDocRepo := newMockSpecialistDocumentRepo()
	uc := NewDissociateDocumentUseCase(specDocRepo)

	err := uc.Execute(context.Background(), "spec-1", "doc-1")

	assert.ErrorIs(t, err, domain.ErrDocumentNotAssociated)
}
