package application

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// --- mock implementations for ConversationEngine ---

type mockConvStateRepo struct {
	state     *domain.ConversationState
	findErr   error
	createErr error
	updateErr error
	updated   *domain.ConversationState
}

func (m *mockConvStateRepo) Create(_ context.Context, s *domain.ConversationState) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.state = s
	return nil
}

func (m *mockConvStateRepo) FindByConversationID(_ context.Context, _ string) (*domain.ConversationState, error) {
	return m.state, m.findErr
}

func (m *mockConvStateRepo) Update(_ context.Context, s *domain.ConversationState) error {
	m.updated = s
	return m.updateErr
}

type mockConfigResolver struct {
	cfg *domain.AIConfig
}

func (m *mockConfigResolver) Resolve(_ context.Context, _ string) *domain.AIConfig {
	return m.cfg
}

type mockAIProvider struct {
	name string
	resp *domain.AIResponse
	err  error
}

func (m *mockAIProvider) GenerateResponse(_ context.Context, _ *domain.AIRequest) (*domain.AIResponse, error) {
	return m.resp, m.err
}

func (m *mockAIProvider) Name() string {
	return m.name
}

type mockMessageSender struct {
	sent    bool
	content string
	err     error
}

func (m *mockMessageSender) SendAIResponse(_ context.Context, _, _, content string) error {
	m.sent = true
	m.content = content
	return m.err
}

type mockLeadUpdater struct {
	scoreUpdated bool
	movedColumn  string
	outcomes     map[string]string
}

func (m *mockLeadUpdater) UpdateLeadScore(_ context.Context, _ string, _ int) error {
	m.scoreUpdated = true
	return nil
}

func (m *mockLeadUpdater) MoveLeadToColumn(_ context.Context, _, columnID string) error {
	m.movedColumn = columnID
	return nil
}

func (m *mockLeadUpdater) SetOutcome(_ context.Context, conversationID, outcome string) error {
	if m.outcomes == nil {
		m.outcomes = make(map[string]string)
	}
	m.outcomes[conversationID] = outcome
	return nil
}

func (m *mockLeadUpdater) GetLeadIDByConversation(_ context.Context, conversationID string) (string, error) {
	// Return a deterministic lead ID derived from the conversation ID so tests
	// can assert that originLeadID != conversationID.
	return "lead-for-" + conversationID, nil
}

func (m *mockLeadUpdater) LastOutcomeFor(conversationID string) string {
	return m.outcomes[conversationID]
}

type mockHandoffActivator struct {
	activatedFor []string
}

func (m *mockHandoffActivator) Activate(_ context.Context, conversationID string) error {
	m.activatedFor = append(m.activatedFor, conversationID)
	return nil
}

func (m *mockHandoffActivator) WasActivatedFor(conversationID string) bool {
	for _, id := range m.activatedFor {
		if id == conversationID {
			return true
		}
	}
	return false
}

// mockSpecialistTenantChecker reports whether a specialist is associated with a
// tenant. Any specialist ID present in `associated` is considered still linked.
type mockSpecialistTenantChecker struct {
	associated map[string]bool
	err        error
}

func (m *mockSpecialistTenantChecker) Exists(_ context.Context, specialistID, _ string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.associated[specialistID], nil
}

// --- helpers ---

func buildEngineFixtures(t *testing.T, state *domain.ConversationState, findErr error, inactiveSpecialists ...string) (
	*ConversationEngine,
	*mockConvStateRepo,
	*mockMessageSender,
) {
	t.Helper()

	inactive := make(map[string]bool, len(inactiveSpecialists))
	for _, id := range inactiveSpecialists {
		inactive[id] = true
	}

	registry := domain.NewProviderRegistry()
	provider := &mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: "Olá, como posso ajudar?", PromptTokens: 10, CompletionTokens: 5},
	}
	registry.Register(provider)

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)

	stateRepo := &mockConvStateRepo{state: state, findErr: findErr}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}
	log := zap.NewNop()

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist, inactive: inactive},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry,
		configResolver,
		stateRepo,
		contextBuilder,
		NewStepEvaluator(),
		NewGuardrailChecker(),
		sender,
		leadUpdater,
		nil,
		false,
		nil,
		0,
		5,
		nil,
		nil,
		nil, nil, nil, // cross-sell: disabled
		log,
	)

	return engine, stateRepo, sender
}

// --- tests ---

func TestConversationEngine_BasicFlow(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	engine, stateRepo, sender := buildEngineFixtures(t, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"})

	require.NoError(t, err)
	assert.True(t, sender.sent, "message should have been sent")
	assert.Equal(t, "Olá, como posso ajudar?", sender.content)
	assert.NotNil(t, stateRepo.updated, "state should have been updated")
}

// TestConversationEngine_ObservesSpecialistResponseDuration confirms every
// HandleMessages call feeds the cross-module SpecialistResponseDuration
// histogram (used by SLO alerts / tenant-wide dashboards).
func TestConversationEngine_ObservesSpecialistResponseDuration(t *testing.T) {
	before := testutil.CollectAndCount(observability.SpecialistResponseDuration)

	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	engine, _, _ := buildEngineFixtures(t, state, nil)

	require.NoError(t, engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"}))

	after := testutil.CollectAndCount(observability.SpecialistResponseDuration)
	assert.GreaterOrEqual(t, after, before, "histogram should keep or add series after HandleMessages")
	assert.GreaterOrEqual(t, after, 1, "{outcome=ok} series must exist after a successful call")
}

// TestConversationEngine_CreatesSpan verifies that HandleMessages emits the
// expected OpenTelemetry span.
func TestConversationEngine_CreatesSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	engine, _, _ := buildEngineFixtures(t, state, nil)

	require.NoError(t, engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"}))

	names := []string{}
	for _, s := range sr.Ended() {
		names = append(names, s.Name())
	}
	assert.Contains(t, names, "ai.usecase.respond", "respond span should be present")
}

func TestConversationEngine_HandoffActive_NoMessageSent(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	state.ActivateHandoff()

	engine, _, sender := buildEngineFixtures(t, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"})

	require.NoError(t, err)
	assert.False(t, sender.sent, "message should NOT be sent when handoff is active")
}

func TestConversationEngine_StateNotFound_CreatesNew(t *testing.T) {
	engine, stateRepo, sender := buildEngineFixtures(t, nil, domain.ErrConversationStateNotFound)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-new", "spec-1", "", []string{"nova conversa"})

	require.NoError(t, err)
	assert.True(t, sender.sent, "message should have been sent for new state")
	assert.NotNil(t, stateRepo.state, "state should have been created")
}

func TestConversationEngine_GuardrailViolation_UsesFallback(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")

	registry := domain.NewProviderRegistry()
	provider := &mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: "Vamos falar sobre politica..."},
	}
	registry.Register(provider)

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	log := zap.NewNop()

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{},
		&mockGuardrailFinder{guardrails: []specDomain.Guardrail{
			{ID: "g1", Type: specDomain.GuardrailTypeForbiddenTopics, Rule: "politica", Message: "Fora do escopo.", Active: true},
		}},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, nil, nil, false, nil, 0, 5, nil, nil,
		nil, nil, nil, // cross-sell: disabled
		log,
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"oi"})

	require.NoError(t, err)
	assert.True(t, sender.sent)
	assert.Equal(t, "Fora do escopo.", sender.content)
}

func TestConversationEngine_StepCompleted_RuleBased(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")

	registry := domain.NewProviderRegistry()
	provider := &mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: "Obrigado pelo número!"},
	}
	registry.Register(provider)

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}
	log := zap.NewNop()

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{steps: []specDomain.Step{
			{ID: "step-1", Text: "Informe seu CPF.", DataType: specDomain.StepDataTypeNumber, Score: 10, TargetColumnID: "col-2"},
		}},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater, nil, false, nil, 0, 5, nil, nil,
		nil, nil, nil, // cross-sell: disabled
		log,
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"42"})

	require.NoError(t, err)
	assert.True(t, sender.sent)
	assert.Equal(t, 1, state.CurrentStepIndex, "step index should have advanced")
	assert.Equal(t, 10, state.AccumulatedScore)
	assert.True(t, leadUpdater.scoreUpdated)
	assert.Equal(t, "col-2", leadUpdater.movedColumn)
}

type mockScoringFinder struct {
	config *specDomain.ScoringConfig
	err    error
}

func (m *mockScoringFinder) FindBySpecialistID(_ context.Context, _ string) (*specDomain.ScoringConfig, error) {
	return m.config, m.err
}

func buildScoringEngine(
	t *testing.T,
	steps []specDomain.Step,
	scoring *specDomain.ScoringConfig,
	state *domain.ConversationState,
	handoffActivator HandoffActivator,
) (*ConversationEngine, *mockLeadUpdater) {
	t.Helper()

	registry := domain.NewProviderRegistry()
	registry.Register(&mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: "ok"},
	})

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{steps: steps},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	var scoringFinder ScoringConfigFinder
	if scoring != nil {
		scoringFinder = &mockScoringFinder{config: scoring}
	}

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater,
		nil, false, nil, 0, 5,
		scoringFinder,
		handoffActivator,
		nil, nil, nil, // cross-sell: disabled
		zap.NewNop(),
	)
	return engine, leadUpdater
}

func TestConversationEngine_ScoringQualifies_MovesLeadToQualifiedColumn(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Confirma?", DataType: specDomain.StepDataTypeSelection, Score: 60, TargetColumnID: ""},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            60,
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}

	engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, 60, state.AccumulatedScore)
	assert.Equal(t, "col-qualified", leadUpdater.movedColumn, "lead must move to qualified column when score crosses threshold")
}

func TestConversationEngine_AllStepsDoneUnderThreshold_MovesLeadToDisqualifiedColumn(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Confirma?", DataType: specDomain.StepDataTypeSelection, Score: 10, TargetColumnID: ""},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            60,
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}

	engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, 10, state.AccumulatedScore)
	assert.Equal(t, "col-disqualified", leadUpdater.movedColumn, "lead must move to disqualified column when all steps are done and score is below threshold")
}

func TestConversationEngine_StepTargetWinsOverScoring(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Confirma?", DataType: specDomain.StepDataTypeSelection, Score: 60, TargetColumnID: "col-step-explicit"},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:      "spec-1",
		Threshold:         60,
		QualifiedColumnID: "col-qualified",
	}

	engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, "col-step-explicit", leadUpdater.movedColumn, "explicit step target must override scoring threshold")
}

func TestConversationEngine_LLMDisqualifies_MovesLeadToDisqualifiedColumn(t *testing.T) {
	// Mid-flow veto: two steps remain, score below threshold, AND the step has
	// an explicit target_column_id that would normally route the lead forward.
	// The LLM's `disqualified:true` flag must override the step target and move
	// the lead straight to the disqualified column.
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Conte sua situacao.", DataType: specDomain.StepDataTypeFreeText, Score: 10, TargetColumnID: "col-next-stage"},
		{ID: "step-2", Text: "Mais um detalhe.", DataType: specDomain.StepDataTypeFreeText, Score: 10, TargetColumnID: ""},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            60,
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}

	registry := domain.NewProviderRegistry()
	registry.Register(&mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: `Entendi, infelizmente nao conseguimos seguir.<!--STEP_META:{"step_completed":true,"collected_data":"nunca contribuiu","score":0,"disqualified":true}-->`},
	})

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{steps: steps},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater,
		nil, false, nil, 0, 5,
		&mockScoringFinder{config: scoring},
		nil,
		nil, nil, nil, // cross-sell: disabled
		zap.NewNop(),
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"nunca contribui"})

	require.NoError(t, err)
	assert.Equal(t, "col-disqualified", leadUpdater.movedColumn, "LLM disqualified flag must override and move lead to disqualified column")
}

func TestConversationEngine_LLMDisqualifies_NoScoringConfig_NoMove(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Conte sua situacao.", DataType: specDomain.StepDataTypeFreeText, Score: 10},
	}

	registry := domain.NewProviderRegistry()
	registry.Register(&mockAIProvider{
		name: "openai",
		resp: &domain.AIResponse{Content: `Ok.<!--STEP_META:{"step_completed":true,"collected_data":"x","score":0,"disqualified":true}-->`},
	})

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{steps: steps},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	// No scoring finder → disqualify flag has no target to move to.
	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater,
		nil, false, nil, 0, 5,
		nil,
		nil,
		nil, nil, nil, // cross-sell: disabled
		zap.NewNop(),
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"x"})

	require.NoError(t, err)
	assert.Equal(t, "", leadUpdater.movedColumn, "no move when scoring finder is not configured")
}

func TestConversationEngine_MidFlow_UnderThreshold_NoScoringMove(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "A?", DataType: specDomain.StepDataTypeSelection, Score: 20, TargetColumnID: ""},
		{ID: "step-2", Text: "B?", DataType: specDomain.StepDataTypeSelection, Score: 60, TargetColumnID: ""},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            60,
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}

	engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, 20, state.AccumulatedScore)
	assert.Equal(t, "", leadUpdater.movedColumn, "no move when score is below threshold and there are still steps pending")
}

func TestHandleMessages_ResetCommandEnabled_TriggersReset(t *testing.T) {
	// Reuse stubs from reset_conversation_test.go (same package).
	existing, err := domain.NewConversationState("id", "conv-1", "spec-1")
	require.NoError(t, err)
	existing.AdvanceStep("nome", "Maria", 10)
	stateRepo := &stubStateRepo{state: existing}

	sender := &stubSender{}
	resetUC := NewResetConversationUseCase(
		stateRepo,
		stubEntryColumn{id: "col-entry"},
		&stubLeadResetter{},
		sender,
		zap.NewNop(),
	)

	engine := NewConversationEngine(
		nil, nil, stateRepo, nil, nil, nil, sender, nil,
		resetUC, true, nil, 0, 5, nil, nil,
		nil, nil, nil, // cross-sell: disabled
		zap.NewNop(),
	)

	err = engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"/reset"})
	require.NoError(t, err)
	assert.Equal(t, 0, stateRepo.state.CurrentStepIndex, "state should be zeroed")
	assert.Equal(t, 0, stateRepo.state.AccumulatedScore, "score should be zeroed")
	assert.Contains(t, sender.lastContent, "reiniciada", "should send reset confirmation")
}

func TestHandleMessages_ResetCommandDisabled_SkipsResetInterception(t *testing.T) {
	// When the flag is off, /reset should flow through the normal path: the
	// engine calls the provider and sends the provider's AI response back —
	// NOT the reset confirmation message. We prove this by using the full
	// fixture stack and asserting the sender receives the provider's content.
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	engine, stateRepo, sender := buildEngineFixtures(t, state, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"/reset"})
	require.NoError(t, err)
	assert.True(t, sender.sent, "normal path should have sent a message")
	assert.Equal(t, "Olá, como posso ajudar?", sender.content,
		"should be the provider's response, not the reset confirmation")
	assert.NotContains(t, sender.content, "reiniciada",
		"reset interception must NOT run when flag is disabled")
	assert.NotNil(t, stateRepo.updated,
		"normal engine flow should have updated state (proving the shortcut was skipped)")
}

func TestConversationEngine_ProviderError(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")

	registry := domain.NewProviderRegistry()
	provider := &mockAIProvider{
		name: "openai",
		err:  errors.New("provider unavailable"),
	}
	registry.Register(provider)

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	log := zap.NewNop()

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e uma advogada.")
	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, nil, nil, false, nil, 0, 5, nil, nil,
		nil, nil, nil, // cross-sell: disabled
		log,
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"oi"})

	require.Error(t, err)
	assert.False(t, sender.sent)
}

func TestConversationEngine_ScoringInHumanZone_PausesAIAndMovesLead(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Confirma?", DataType: specDomain.StepDataTypeSelection, Score: 60, TargetColumnID: ""},
	}
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            80,
		ThresholdHumanoMin:   50,
		HumanColumnID:        "col-h",
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}

	fakeHandoff := &mockHandoffActivator{}
	engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, fakeHandoff)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, 60, state.AccumulatedScore)
	assert.True(t, fakeHandoff.WasActivatedFor("conv-1"), "handoff must be activated for the conversation")
	assert.Equal(t, "col-h", leadUpdater.movedColumn, "lead must move to human column when score is in human zone")
}

func TestConversationEngine_PersistsOutcomeInLead(t *testing.T) {
	scoring := &specDomain.ScoringConfig{
		SpecialistID:         "spec-1",
		Threshold:            80,
		ThresholdHumanoMin:   50,
		HumanColumnID:        "col-h",
		QualifiedColumnID:    "col-qualified",
		DisqualifiedColumnID: "col-disqualified",
	}
	steps := []specDomain.Step{
		{ID: "step-1", Text: "Confirma?", DataType: specDomain.StepDataTypeSelection, Score: 0, TargetColumnID: ""},
	}

	t.Run("aprovado", func(t *testing.T) {
		state, _ := domain.NewConversationState("s-1", "conv-aprovado", "spec-1")
		state.AccumulatedScore = 80 // already at threshold before message
		engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

		err := engine.HandleMessages(context.Background(), "tenant-1", "conv-aprovado", "spec-1", "", []string{"sim"})

		require.NoError(t, err)
		assert.Equal(t, "aprovado", leadUpdater.LastOutcomeFor("conv-aprovado"))
	})

	t.Run("humano", func(t *testing.T) {
		state, _ := domain.NewConversationState("s-1", "conv-humano", "spec-1")
		state.AccumulatedScore = 60 // in humano zone [50, 80)
		engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

		err := engine.HandleMessages(context.Background(), "tenant-1", "conv-humano", "spec-1", "", []string{"sim"})

		require.NoError(t, err)
		assert.Equal(t, "humano", leadUpdater.LastOutcomeFor("conv-humano"))
	})

	t.Run("reprovado", func(t *testing.T) {
		state, _ := domain.NewConversationState("s-1", "conv-reprovado", "spec-1")
		state.AccumulatedScore = 10 // below humano min
		engine, leadUpdater := buildScoringEngine(t, steps, scoring, state, nil)

		err := engine.HandleMessages(context.Background(), "tenant-1", "conv-reprovado", "spec-1", "", []string{"sim"})

		require.NoError(t, err)
		assert.Equal(t, "reprovado", leadUpdater.LastOutcomeFor("conv-reprovado"))
	})
}

// ─── cross-sell mocks ────────────────────────────────────────────────────────

type mockCrossSellRuleRepo struct {
	rules   []*specDomain.CrossSellRule
	byID    map[string]*specDomain.CrossSellRule
	saved   []*specDomain.CrossSellRule
	deleted []string
}

func newMockCrossSellRuleRepo(rules ...*specDomain.CrossSellRule) *mockCrossSellRuleRepo {
	m := &mockCrossSellRuleRepo{
		rules: rules,
		byID:  make(map[string]*specDomain.CrossSellRule),
	}
	for _, r := range rules {
		m.byID[r.ID] = r
	}
	return m
}

func (m *mockCrossSellRuleRepo) Save(_ context.Context, r *specDomain.CrossSellRule) error {
	m.saved = append(m.saved, r)
	return nil
}
func (m *mockCrossSellRuleRepo) FindByID(_ context.Context, id string) (*specDomain.CrossSellRule, error) {
	r, ok := m.byID[id]
	if !ok {
		return nil, specDomain.ErrCrossSellRuleNotFound
	}
	return r, nil
}
func (m *mockCrossSellRuleRepo) ListBySpecialistID(_ context.Context, _ string) ([]*specDomain.CrossSellRule, error) {
	return m.rules, nil
}
func (m *mockCrossSellRuleRepo) ListActiveBySpecialistOrdered(_ context.Context, _ string) ([]*specDomain.CrossSellRule, error) {
	var out []*specDomain.CrossSellRule
	for _, r := range m.rules {
		if r.Ativo {
			out = append(out, r)
		}
	}
	return out, nil
}
func (m *mockCrossSellRuleRepo) Delete(_ context.Context, id string) error {
	m.deleted = append(m.deleted, id)
	return nil
}

type mockProductSpecialistResolver struct {
	specialistID, funnelID, columnID string
}

func (m *mockProductSpecialistResolver) FindSpecialistByProduct(_ context.Context, _ string) (string, string, string, error) {
	return m.specialistID, m.funnelID, m.columnID, nil
}

type mockLeadFactory struct {
	createdLeadID        string
	capturedOriginLeadID string
	calls                int
}

func (m *mockLeadFactory) CreateForCrossSell(_ context.Context, originLeadID, _, _, _, _ string) (string, error) {
	m.calls++
	m.capturedOriginLeadID = originLeadID
	if m.createdLeadID == "" {
		m.createdLeadID = "new-lead-id"
	}
	return m.createdLeadID, nil
}

type mockConversationMover struct {
	migratedTo     string
	pendingSet     string
	pendingCleared bool
}

func (m *mockConversationMover) MigrateSpecialist(_ context.Context, _, newSpecialistID string) error {
	m.migratedTo = newSpecialistID
	return nil
}
func (m *mockConversationMover) SetPendingCrossSell(_ context.Context, _, ruleID string) error {
	m.pendingSet = ruleID
	return nil
}
func (m *mockConversationMover) ClearPendingCrossSell(_ context.Context, _ string) error {
	m.pendingCleared = true
	return nil
}

type mockProductNameLookup struct {
	name string
}

func (m *mockProductNameLookup) Name(_ context.Context, _ string) (string, error) {
	if m.name == "" {
		return "Produto B", nil
	}
	return m.name, nil
}

// buildCrossSellEngine is a test helper that wires an engine with cross-sell enabled.
func buildCrossSellEngine(
	t *testing.T,
	state *domain.ConversationState,
	specialist *specDomain.Specialist,
	rules []*specDomain.CrossSellRule,
	mover *mockConversationMover,
	provider *mockAIProvider,
) (*ConversationEngine, *mockMessageSender, *mockLeadFactory, *mockConversationMover) {
	t.Helper()

	registry := domain.NewProviderRegistry()
	if provider == nil {
		provider = &mockAIProvider{
			name: "openai",
			resp: &domain.AIResponse{Content: "resposta do LLM"},
		}
	}
	registry.Register(provider)

	cfg, _ := domain.NewAIConfig("cfg-1", specialist.ID, "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}
	log := zap.NewNop()

	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	ruleRepo := newMockCrossSellRuleRepo(rules...)
	if mover == nil {
		mover = &mockConversationMover{}
	}
	lf := &mockLeadFactory{}
	executor := NewCrossSellExecutor(
		&mockProductSpecialistResolver{specialistID: "spec-new", funnelID: "funnel-1", columnID: "col-1"},
		lf,
		mover,
		leadUpdater,
		sender,
		&mockProductNameLookup{name: "Produto B"},
	)

	engine := NewConversationEngine(
		registry, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater,
		nil, false, nil, 0, 5,
		nil, nil,
		ruleRepo, NewCrossSellRuleEvaluator(), executor,
		log,
	)
	return engine, sender, lf, mover
}

// TestConversationEngine_KeywordRuleTriggersBeforeLLM verifies that when a cross-sell
// keyword rule matches, the executor fires and the LLM provider is NOT called.
func TestConversationEngine_KeywordRuleTriggersBeforeLLM(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e advogada.")
	require.NoError(t, specialist.EnableCrossSell(specDomain.CrossSellModeSilent, ""))

	rule, _ := specDomain.NewCrossSellRule("rule-1", "spec-1", 0,
		specDomain.CrossSellTriggerKeyword,
		specDomain.KeywordTrigger{Termos: []string{"trabalhista"}},
		"prod-2",
	)

	// Use a provider that panics if GenerateResponse is called — proves LLM is NOT invoked.
	panicProvider := &panicAIProvider{name: "openai"}
	registry2 := domain.NewProviderRegistry()
	registry2.Register(panicProvider)

	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
	stateRepo := &mockConvStateRepo{state: state}
	configResolver := &mockConfigResolver{cfg: cfg}
	sender := &mockMessageSender{}
	leadUpdater := &mockLeadUpdater{}
	mover := &mockConversationMover{}
	lf := &mockLeadFactory{}

	contextBuilder := NewContextBuilder(
		&mockSpecialistFinder{specialist: specialist},
		&mockStepFinder{},
		&mockGuardrailFinder{},
		&mockDocumentFetcher{},
		&mockProductInfoFinder{},
		&mockMessageHistoryFinder{},
		nil,
	)

	ruleRepo := newMockCrossSellRuleRepo(rule)
	executor := NewCrossSellExecutor(
		&mockProductSpecialistResolver{specialistID: "spec-new", funnelID: "funnel-1", columnID: "col-1"},
		lf,
		mover,
		leadUpdater,
		sender,
		&mockProductNameLookup{name: "Produto B"},
	)

	engine := NewConversationEngine(
		registry2, configResolver, stateRepo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), sender, leadUpdater,
		nil, false, nil, 0, 5,
		nil, nil,
		ruleRepo, NewCrossSellRuleEvaluator(), executor,
		zap.NewNop(),
	)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"tenho duvida trabalhista"})

	// LLM was NOT invoked (silent mode: executor.Execute called instead).
	require.NoError(t, err)
	assert.Equal(t, 1, lf.calls, "LeadFactory should have been called once for the cross-sell transition")
	assert.Equal(t, "spec-new", mover.migratedTo, "conversation must be migrated to the new specialist")
	// Bug 1 regression: originLeadID must be the lead's ID, NOT the conversation ID.
	assert.Equal(t, "lead-for-conv-1", lf.capturedOriginLeadID, "originLeadID must be lead.ID, not conversationID")
	assert.NotEqual(t, "conv-1", lf.capturedOriginLeadID, "originLeadID must not equal conversationID")
}

// panicAIProvider panics if GenerateResponse is called — used to assert LLM is NOT invoked.
type panicAIProvider struct {
	name string
}

func (p *panicAIProvider) Name() string { return p.name }
func (p *panicAIProvider) GenerateResponse(_ context.Context, _ *domain.AIRequest) (*domain.AIResponse, error) {
	panic("LLM provider must NOT be called when a cross-sell rule matches before it")
}

// TestConversationEngine_ConfirmMode_SetsPendingAndSendsQuestion verifies that when
// confirm-mode fires, a question is sent and PendingCrossSellRuleID is persisted.
func TestConversationEngine_ConfirmMode_SetsPendingAndSendsQuestion(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e advogada.")
	require.NoError(t, specialist.EnableCrossSell(specDomain.CrossSellModeConfirm, ""))

	rule, _ := specDomain.NewCrossSellRule("rule-confirm", "spec-1", 0,
		specDomain.CrossSellTriggerKeyword,
		specDomain.KeywordTrigger{Termos: []string{"trabalhista"}},
		"prod-2",
	)

	engine, sender, _, mover := buildCrossSellEngine(t, state, specialist, []*specDomain.CrossSellRule{rule}, nil, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"tenho duvida trabalhista"})

	require.NoError(t, err)
	assert.True(t, sender.sent, "confirmation question must be sent")
	assert.Contains(t, sender.content, "Produto B", "question must mention the product name")
	assert.Equal(t, "rule-confirm", mover.pendingSet, "pending rule ID must be stored in conversation state")
}

// TestConversationEngine_ConfirmMode_PositiveReply_CompletesTransition verifies that
// when a pending confirm rule exists and the user replies affirmatively, CompleteTransition fires.
// When the specialist persisted on an existing conversation was removed from the
// tenant (dissociated in admin), the next message must re-adopt the router's fresh
// specialist and persist it, so the conversation stops "falling into the first".
func TestConversationEngine_HealsSpecialistWhenRemovedFromTenant(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-old")
	engine, stateRepo, sender := buildEngineFixtures(t, state, nil)
	// spec-old is no longer associated with the tenant; router now resolves spec-1.
	engine.SetSpecialistTenantChecker(&mockSpecialistTenantChecker{
		associated: map[string]bool{"spec-1": true},
	})

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"})

	require.NoError(t, err)
	require.NotNil(t, stateRepo.updated, "state must be persisted")
	assert.Equal(t, "spec-1", stateRepo.updated.SpecialistID,
		"conversation must migrate to the router-resolved specialist")
	assert.True(t, sender.sent)
}

// When the specialist persisted on an existing conversation is deactivated in admin
// (still associated but inactive), the next message must re-adopt the router's
// specialist — deactivating stops the conversation from using the old specialist.
func TestConversationEngine_HealsSpecialistWhenDeactivated(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-old")
	// spec-old is still associated with the tenant but has been deactivated.
	engine, stateRepo, _ := buildEngineFixtures(t, state, nil, "spec-old")
	engine.SetSpecialistTenantChecker(&mockSpecialistTenantChecker{
		associated: map[string]bool{"spec-old": true, "spec-1": true},
	})

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"})

	require.NoError(t, err)
	require.NotNil(t, stateRepo.updated, "state must be persisted")
	assert.Equal(t, "spec-1", stateRepo.updated.SpecialistID,
		"a deactivated specialist must be replaced by the router-resolved one")
}

// An explicit switch (cross-sell/tool/automation) leaves the conversation on a
// specialist that IS still associated with the tenant. Even if the router would
// pick a different one, the engine must NOT override it — the switch sticks.
func TestConversationEngine_KeepsSpecialistWhenStillAssociated(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-switched")
	engine, stateRepo, _ := buildEngineFixtures(t, state, nil)
	// spec-switched remains associated; router would pick spec-1.
	engine.SetSpecialistTenantChecker(&mockSpecialistTenantChecker{
		associated: map[string]bool{"spec-switched": true, "spec-1": true},
	})

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"})

	require.NoError(t, err)
	require.NotNil(t, stateRepo.updated, "state must be persisted")
	assert.Equal(t, "spec-switched", stateRepo.updated.SpecialistID,
		"an explicitly-switched specialist that is still associated must be preserved")
}

func TestConversationEngine_ConfirmMode_PositiveReply_CompletesTransition(t *testing.T) {
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	state.SetPendingCrossSellRuleID("rule-confirm")

	specialist, _ := specDomain.NewSpecialist("spec-1", "Ana", "Advogada", "Voce e advogada.")
	require.NoError(t, specialist.EnableCrossSell(specDomain.CrossSellModeConfirm, ""))

	rule, _ := specDomain.NewCrossSellRule("rule-confirm", "spec-1", 0,
		specDomain.CrossSellTriggerKeyword,
		specDomain.KeywordTrigger{Termos: []string{"trabalhista"}},
		"prod-2",
	)
	mover := &mockConversationMover{}
	engine, _, lf, _ := buildCrossSellEngine(t, state, specialist, []*specDomain.CrossSellRule{rule}, mover, nil)

	err := engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"sim"})

	require.NoError(t, err)
	assert.Equal(t, 1, lf.calls, "new lead must be created for the cross-sell")
	assert.True(t, mover.pendingCleared, "pending state must be cleared after positive reply")
	assert.Equal(t, "spec-new", mover.migratedTo, "conversation must migrate to new specialist")
	// Bug 1 regression: originLeadID must be the lead's ID, NOT the conversation ID.
	assert.Equal(t, "lead-for-conv-1", lf.capturedOriginLeadID, "originLeadID must be lead.ID, not conversationID")
	assert.NotEqual(t, "conv-1", lf.capturedOriginLeadID, "originLeadID must not equal conversationID")
}
