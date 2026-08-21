package application

import (
	"context"
	"testing"
	"time"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock implementations for ContextBuilder ---

type mockSpecialistFinder struct {
	specialist *specDomain.Specialist
	err        error
	// inactive: IDs for which FindByID returns a deactivated copy of specialist.
	inactive map[string]bool
}

func (m *mockSpecialistFinder) FindByID(_ context.Context, id string) (*specDomain.Specialist, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.inactive[id] && m.specialist != nil {
		cp := *m.specialist
		cp.Status = specDomain.SpecialistStatusInactive
		return &cp, nil
	}
	return m.specialist, m.err
}

type mockStepFinder struct {
	steps []specDomain.Step
}

func (m *mockStepFinder) FindBySpecialistID(_ context.Context, _ string) ([]specDomain.Step, error) {
	return m.steps, nil
}

type mockGuardrailFinder struct {
	guardrails []specDomain.Guardrail
}

func (m *mockGuardrailFinder) FindBySpecialistID(_ context.Context, _ string) ([]specDomain.Guardrail, error) {
	return m.guardrails, nil
}

type mockDocumentFetcher struct {
	docs []string
}

func (m *mockDocumentFetcher) FetchDocumentContents(_ context.Context, _ string) ([]string, error) {
	return m.docs, nil
}

type mockProductInfoFinder struct {
	name, description string
	err               error
}

func (m *mockProductInfoFinder) FindProductInfo(_ context.Context, _ string) (string, string, error) {
	return m.name, m.description, m.err
}

type mockMessageHistoryFinder struct {
	messages []HistoryMessage
}

func (m *mockMessageHistoryFinder) FindHistory(_ context.Context, _ string, _ int) ([]HistoryMessage, error) {
	return m.messages, nil
}

// --- helper ---

func buildTestState() *domain.ConversationState {
	state, _ := domain.NewConversationState("state-1", "conv-1", "spec-1")
	return state
}

func buildTestSpecialist() *specDomain.Specialist {
	s, _ := specDomain.NewSpecialist("spec-1", "Joao", "Especialista em vendas", "Voce e um assistente de vendas.")
	return s
}

// --- tests ---

func TestContextBuilder_FullBuild(t *testing.T) {
	state := buildTestState()

	now := time.Now()
	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{steps: []specDomain.Step{
			{ID: "step-1", Text: "Qual o seu nome?", DataType: specDomain.StepDataTypeNumber, Score: 5},
		}},
		&mockGuardrailFinder{guardrails: []specDomain.Guardrail{
			{ID: "g1", Type: specDomain.GuardrailTypeForbiddenTopics, Rule: "politica", Message: "Fora do escopo", Active: true},
			{ID: "g2", Type: specDomain.GuardrailTypeForbiddenTopics, Rule: "religiao", Message: "Fora do escopo", Active: false},
		}},
		&mockDocumentFetcher{docs: []string{"Documento A", "Documento B"}},
		&mockProductInfoFinder{name: "Seguro Auto", description: "Protecao total para seu veiculo."},
		&mockMessageHistoryFinder{messages: []HistoryMessage{
			{Role: domain.RoleUser, Content: "oi", Timestamp: now},
		}},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "prod-1", 20)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Contains(t, req.SystemPrompt, "Voce e Joao")
	assert.Contains(t, req.SystemPrompt, "Seguro Auto")
	assert.Contains(t, req.SystemPrompt, "Documento A")
	assert.Contains(t, req.SystemPrompt, "Documento B")
	assert.Contains(t, req.SystemPrompt, "politica")
	assert.NotContains(t, req.SystemPrompt, "religiao") // inactive guardrail
	assert.Contains(t, req.SystemPrompt, "Qual o seu nome?")

	require.Len(t, req.Messages, 1)
	assert.Equal(t, domain.RoleUser, req.Messages[0].Role)
	assert.Equal(t, "oi", req.Messages[0].Content)
}

func TestContextBuilder_BuildWithoutSteps(t *testing.T) {
	state := buildTestState()

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{steps: nil},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "", 10)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.NotContains(t, req.SystemPrompt, "Etapa atual")
	// Falls back to "oi" when no history.
	require.Len(t, req.Messages, 1)
	assert.Equal(t, "oi", req.Messages[0].Content)
}

func TestContextBuilder_FilterMessagesByResumedAt(t *testing.T) {
	state := buildTestState()

	resumeTime := time.Now()
	state.ResumedAt = &resumeTime

	before := resumeTime.Add(-1 * time.Minute)
	after := resumeTime.Add(1 * time.Minute)

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{messages: []HistoryMessage{
			{Role: domain.RoleUser, Content: "mensagem antiga", Timestamp: before},
			{Role: domain.RoleUser, Content: "mensagem nova", Timestamp: after},
		}},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "", 10)
	require.NoError(t, err)

	require.Len(t, req.Messages, 1)
	assert.Equal(t, "mensagem nova", req.Messages[0].Content)
}

func TestContextBuilder_FreeTextStepAddsMetaInstruction(t *testing.T) {
	state := buildTestState()

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{steps: []specDomain.Step{
			{ID: "step-1", Text: "Descreva o problema.", DataType: specDomain.StepDataTypeFreeText, Score: 10},
		}},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "", 10)
	require.NoError(t, err)
	assert.Contains(t, req.SystemPrompt, "STEP_META")
}

func TestContextBuilder_Build_WithToolResolver_PopulatesTools(t *testing.T) {
	state := buildTestState()

	registry := NewToolRegistry()
	registry.Register(newFakeTool("search_leads", domain.ToolCategoryDataQuery))

	finder := &fakeSpecialistToolFinder{
		toolNames: map[string][]string{
			"spec-1": {"search_leads"},
		},
	}

	resolver := NewToolResolver(registry, finder)

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		resolver,
	)

	req, err := builder.Build(context.Background(), state, "", 20)
	require.NoError(t, err)
	require.Len(t, req.Tools, 1)
	assert.Equal(t, "search_leads", req.Tools[0].Name)
}

// The whole point of persisting CollectedData is to stop the specialist from
// re-asking what the client already answered. That only works if the data
// reaches the prompt — it used to be saved and never sent.
func TestContextBuilder_IncludesCollectedDataInPrompt(t *testing.T) {
	state := buildTestState()
	state.CurrentStepIndex = 2
	state.CollectedData = map[string]string{
		"step_0": "68 anos",
		"step_1": "Caxias do Sul/RS",
	}

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{steps: []specDomain.Step{
			{ID: "s1", Text: "Qual a sua idade?", DataType: specDomain.StepDataTypeNumber},
			{ID: "s2", Text: "Em qual cidade voce mora?", DataType: specDomain.StepDataTypeFreeText},
			{ID: "s3", Text: "Voce possui documentos?", DataType: specDomain.StepDataTypeFreeText},
		}},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "", 20)
	require.NoError(t, err)

	assert.Contains(t, req.SystemPrompt, "Dados ja coletados")
	// Question paired with its answer, so the model can tell what is already known.
	assert.Contains(t, req.SystemPrompt, "Qual a sua idade?: 68 anos")
	assert.Contains(t, req.SystemPrompt, "Em qual cidade voce mora?: Caxias do Sul/RS")
	// The pending step must not be listed as already answered.
	assert.NotContains(t, req.SystemPrompt, "Voce possui documentos?: ")
}

func TestContextBuilder_OmitsCollectedDataBlockWhenEmpty(t *testing.T) {
	state := buildTestState()

	builder := NewContextBuilder(
		&mockSpecialistFinder{specialist: buildTestSpecialist()},
		&mockStepFinder{steps: []specDomain.Step{{ID: "s1", Text: "Qual a sua idade?", DataType: specDomain.StepDataTypeNumber}}},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	req, err := builder.Build(context.Background(), state, "", 20)
	require.NoError(t, err)
	assert.NotContains(t, req.SystemPrompt, "Dados ja coletados")
}
