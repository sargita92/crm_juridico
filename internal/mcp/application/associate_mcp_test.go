package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func TestAssociateMcpUseCase_Success(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor MCP", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       []string{"mcp-1"},
	})

	require.NoError(t, err)
	exists, _ := specMcpRepo.Exists(context.Background(), "spec-1", "mcp-1")
	assert.True(t, exists)
}

func TestAssociateMcpUseCase_MultipleMcps(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	mcp1, _ := domain.NewMcpServer("mcp-1", "Servidor 1", "https://mcp1.example.com", "", "")
	mcp2, _ := domain.NewMcpServer("mcp-2", "Servidor 2", "https://mcp2.example.com", "", "")
	mcpRepo.addMcp(mcp1)
	mcpRepo.addMcp(mcp2)

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       []string{"mcp-1", "mcp-2"},
	})

	require.NoError(t, err)
	ids, _ := specMcpRepo.FindMcpIDsBySpecialistID(context.Background(), "spec-1")
	assert.Len(t, ids, 2)
}

func TestAssociateMcpUseCase_SpecialistNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "nonexistent",
		McpIDs:       []string{"mcp-1"},
	})

	assert.ErrorIs(t, err, specialistdomain.ErrSpecialistNotFound)
}

func TestAssociateMcpUseCase_SpecialistInactive(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	_ = spec.Deactivate()
	specRepo.addSpecialist(spec)

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       []string{"mcp-1"},
	})

	assert.ErrorIs(t, err, specialistdomain.ErrSpecialistInactive)
}

func TestAssociateMcpUseCase_McpNotFound(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       []string{"nonexistent"},
	})

	assert.ErrorIs(t, err, domain.ErrMcpNotFound)
}

func TestAssociateMcpUseCase_SkipAlreadyAssociated(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	spec, _ := specialistdomain.NewSpecialist("spec-1", "Especialista", "desc", "prompt")
	specRepo.addSpecialist(spec)

	mcp, _ := domain.NewMcpServer("mcp-1", "Servidor MCP", "https://mcp.example.com", "", "")
	mcpRepo.addMcp(mcp)

	_ = specMcpRepo.Associate(context.Background(), "spec-1", "mcp-1")

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       []string{"mcp-1"},
	})

	require.NoError(t, err)
}

func TestAssociateMcpUseCase_BatchTooLarge(t *testing.T) {
	specRepo := newMockSpecialistRepo()
	mcpRepo := newMockMcpServerRepo()
	specMcpRepo := newMockSpecialistMcpRepo()
	uc := NewAssociateMcpUseCase(specRepo, mcpRepo, specMcpRepo)

	ids := make([]string, MaxAssociationBatchSize+1)
	for i := range ids {
		ids[i] = "mcp-" + string(rune('0'+i))
	}

	err := uc.Execute(context.Background(), AssociateMcpInput{
		SpecialistID: "spec-1",
		McpIDs:       ids,
	})

	assert.ErrorIs(t, err, ErrAssociationBatchTooLarge)
}
