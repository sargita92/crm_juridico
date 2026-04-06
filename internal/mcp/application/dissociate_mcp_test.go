package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
)

func TestDissociateMcpUseCase_Success(t *testing.T) {
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewDissociateMcpUseCase(specMcpRepo)

	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")

	err := uc.Execute(context.Background(), "spec-1", "mcp-1")

	require.NoError(t, err)
	exists, _ := specMcpRepo.Exists(context.Background(), "spec-1", "mcp-1")
	assert.False(t, exists)
}

func TestDissociateMcpUseCase_NotAssociated(t *testing.T) {
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewDissociateMcpUseCase(specMcpRepo)

	err := uc.Execute(context.Background(), "spec-1", "mcp-1")

	assert.ErrorIs(t, err, domain.ErrMcpNotAssociated)
}
