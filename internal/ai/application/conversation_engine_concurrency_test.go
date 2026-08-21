package application

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// slowProvider blocks for a beat inside GenerateResponse and records the highest
// number of callers that were inside it at once.
type slowProvider struct {
	delay   time.Duration
	inside  atomic.Int32
	maxSeen atomic.Int32
}

func (p *slowProvider) Name() string { return "openai" }

func (p *slowProvider) GenerateResponse(_ context.Context, _ *domain.AIRequest) (*domain.AIResponse, error) {
	n := p.inside.Add(1)
	for {
		max := p.maxSeen.Load()
		if n <= max || p.maxSeen.CompareAndSwap(max, n) {
			break
		}
	}
	time.Sleep(p.delay)
	p.inside.Add(-1)
	return &domain.AIResponse{Content: "resposta", PromptTokens: 1, CompletionTokens: 1}, nil
}

// countingStateRepo mimics the real read-modify-write cycle: every caller gets
// its own copy of the row, exactly like two goroutines each doing a SELECT.
type countingStateRepo struct {
	mu     sync.Mutex
	stored domain.ConversationState
	writes int
}

func (r *countingStateRepo) Create(_ context.Context, s *domain.ConversationState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stored = *s
	return nil
}

func (r *countingStateRepo) FindByConversationID(_ context.Context, _ string) (*domain.ConversationState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	snapshot := r.stored
	snapshot.CollectedData = make(map[string]string, len(r.stored.CollectedData))
	for k, v := range r.stored.CollectedData {
		snapshot.CollectedData[k] = v
	}
	return &snapshot, nil
}

func (r *countingStateRepo) Update(_ context.Context, s *domain.ConversationState) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stored = *s
	r.writes++
	return nil
}

// nopSender replaces mockMessageSender here: that shared mock writes unguarded
// fields, which the race detector flags as soon as conversations run in parallel.
type nopSender struct{}

func (nopSender) SendAIResponse(_ context.Context, _, _, _ string) error { return nil }

func buildConcurrencyEngine(t *testing.T, provider domain.AIProvider, repo domain.ConversationStateRepository) *ConversationEngine {
	t.Helper()

	registry := domain.NewProviderRegistry()
	registry.Register(provider)
	cfg, _ := domain.NewAIConfig("cfg-1", "spec-1", "openai", "gpt-4", 0.7, 1024, 0)
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

	return NewConversationEngine(
		registry, &mockConfigResolver{cfg: cfg}, repo, contextBuilder,
		NewStepEvaluator(), NewGuardrailChecker(), &nopSender{}, &mockLeadUpdater{},
		nil, false, nil, 0, 5, nil, nil,
		nil, nil, nil,
		zap.NewNop(),
	)
}

// The debounce window is ~3s while an LLM turn takes 5-20s, so a client who
// sends "está aí?" mid-call triggers a second HandleMessages for the same
// conversation. Both would load the same ConversationState and both would write
// it back — a lost update that rewinds the conversation to an earlier step.
// Turns on the same conversation must therefore run one at a time.
func TestConversationEngine_SerializesTurnsPerConversation(t *testing.T) {
	provider := &slowProvider{delay: 60 * time.Millisecond}
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	repo := &countingStateRepo{stored: *state}
	engine := buildConcurrencyEngine(t, provider, repo)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, engine.HandleMessages(context.Background(), "tenant-1", "conv-1", "spec-1", "", []string{"olá"}))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), provider.maxSeen.Load(),
		"two turns on the same conversation ran concurrently — state writes will race")
	assert.Equal(t, 2, repo.writes, "both turns should still complete, just serialized")
}

// Serializing per conversation must not serialize the whole tenant: distinct
// conversations have independent state and have to keep running in parallel.
func TestConversationEngine_DifferentConversationsRunInParallel(t *testing.T) {
	provider := &slowProvider{delay: 60 * time.Millisecond}
	state, _ := domain.NewConversationState("s-1", "conv-1", "spec-1")
	repo := &countingStateRepo{stored: *state}
	engine := buildConcurrencyEngine(t, provider, repo)

	var wg sync.WaitGroup
	for _, convID := range []string{"conv-1", "conv-2", "conv-3"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, engine.HandleMessages(context.Background(), "tenant-1", convID, "spec-1", "", []string{"olá"}))
		}()
	}
	wg.Wait()

	assert.Greater(t, provider.maxSeen.Load(), int32(1),
		"different conversations must not block each other")
}

func TestKeyedMutex_ReleasesEntriesWhenIdle(t *testing.T) {
	km := newKeyedMutex()

	unlock := km.Lock("a")
	unlock()

	km.mu.Lock()
	defer km.mu.Unlock()
	require.Empty(t, km.locks, "idle keys must be evicted so the map does not grow per conversation")
}
