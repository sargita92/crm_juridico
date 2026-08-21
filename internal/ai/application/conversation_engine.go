package application

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	domain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// aiUnavailableFallbackMessage is sent when the provider call fails so the
// conversation does not go silent on the client.
// ponytail: one fixed string — make it per-specialist only if a tenant asks.
const aiUnavailableFallbackMessage = "Tive uma instabilidade aqui, mas já estou retomando seu atendimento. Vamos continuar."

// ConfigResolver resolves the AI configuration for a given specialist.
type ConfigResolver interface {
	Resolve(ctx context.Context, specialistID string) *domain.AIConfig
}

// MessageSender sends an AI-generated response back to the conversation.
type MessageSender interface {
	SendAIResponse(ctx context.Context, tenantID, conversationID, content string) error
}

// LeadUpdater updates lead scoring and column position.
type LeadUpdater interface {
	UpdateLeadScore(ctx context.Context, conversationID string, score int) error
	MoveLeadToColumn(ctx context.Context, conversationID, columnID string) error
	SetOutcome(ctx context.Context, conversationID string, outcome string) error
	// GetLeadIDByConversation resolves the lead ID for a given conversation ID.
	// Used to pass the correct originLeadID to CrossSellExecutor instead of conversationID.
	GetLeadIDByConversation(ctx context.Context, conversationID string) (string, error)
}

// ScoringConfigFinder loads the scoring configuration for a specialist. When
// supplied, the engine promotes a lead to QualifiedColumnID once its score
// crosses Threshold, and demotes it to DisqualifiedColumnID when all steps are
// completed below the threshold.
type ScoringConfigFinder interface {
	FindBySpecialistID(ctx context.Context, specialistID string) (*specDomain.ScoringConfig, error)
}

// HandoffActivator activates a human-agent handoff for a conversation.
// The interface decouples the engine from the concrete ActivateHandoffUseCase.
type HandoffActivator interface {
	Activate(ctx context.Context, conversationID string) error
}

// SpecialistTenantChecker reports whether a specialist is still associated with a
// tenant. It enables the engine to self-heal a conversation whose specialist was
// removed from the tenant in admin: on the next message the conversation re-adopts
// the router's freshly-resolved specialist instead of staying stuck on the old one.
type SpecialistTenantChecker interface {
	Exists(ctx context.Context, specialistID, tenantID string) (bool, error)
}

// ConversationEngine orchestrates the AI conversation flow for a lead.
type ConversationEngine struct {
	providerRegistry    *domain.ProviderRegistry
	configResolver      ConfigResolver
	stateRepo           domain.ConversationStateRepository
	contextBuilder      *ContextBuilder
	stepEvaluator       *StepEvaluator
	guardrailChecker    *GuardrailChecker
	messageSender       MessageSender
	leadUpdater         LeadUpdater
	resetUC             *ResetConversationUseCase
	resetCommandEnabled bool
	toolRegistry        *ToolRegistry
	toolResultMaxLength int
	toolLoopMaxIter     int
	scoringFinder       ScoringConfigFinder
	handoffActivator    HandoffActivator
	// cross-sell fields — all optional (nil-safe); when nil, cross-sell logic is skipped.
	crossSellRuleRepo  specDomain.CrossSellRuleRepository
	crossSellEvaluator *CrossSellRuleEvaluator
	crossSellExecutor  *CrossSellExecutor
	// specialistTenantChecker is optional (nil-safe); when set, the engine heals a
	// conversation whose specialist was dissociated from its tenant by re-adopting
	// the router's freshly-resolved specialist. Nil disables healing.
	specialistTenantChecker SpecialistTenantChecker
	// turnLock serializes turns per conversation. Every path into
	// HandleMessages (debounce callback, playground, cross-sell follow-up) is a
	// read-modify-write on ConversationState, so two overlapping turns lose one
	// another's writes and rewind the conversation to an earlier step.
	turnLock *keyedMutex
	log      *zap.Logger
}

// SetSpecialistTenantChecker installs the optional checker that enables persona
// self-heal when a conversation's specialist is dissociated from its tenant.
// Passing nil (the default) disables healing and preserves prior behavior.
func (e *ConversationEngine) SetSpecialistTenantChecker(c SpecialistTenantChecker) {
	e.specialistTenantChecker = c
}

// personaStillValid reports whether the specialist persisted on a conversation is
// still a legitimate choice: associated with the tenant AND active. On any lookup
// error (or when no checker is wired) it returns true, so the persisted specialist
// is kept — healing only happens on a confirmed removal/deactivation, never on noise.
func (e *ConversationEngine) personaStillValid(ctx context.Context, specialistID, tenantID string) bool {
	if e.specialistTenantChecker == nil {
		return true
	}
	associated, err := e.specialistTenantChecker.Exists(ctx, specialistID, tenantID)
	if err != nil {
		return true
	}
	if !associated {
		return false
	}
	s, err := e.contextBuilder.SpecialistFinder.FindByID(ctx, specialistID)
	if err != nil || s == nil {
		return true
	}
	return s.IsActive()
}

// NewConversationEngine creates a ConversationEngine with all required dependencies.
// scoringFinder is optional: when nil, scoring-based column movement is disabled
// and only explicit step.TargetColumnID drives column transitions.
// handoffActivator is optional: when nil, human-outcome handoff activation is skipped.
// crossSellRuleRepo, crossSellEvaluator, crossSellExecutor are optional (nil-safe);
// when all three are non-nil the engine evaluates cross-sell rules before invoking the LLM.
func NewConversationEngine(
	providerRegistry *domain.ProviderRegistry,
	configResolver ConfigResolver,
	stateRepo domain.ConversationStateRepository,
	contextBuilder *ContextBuilder,
	stepEvaluator *StepEvaluator,
	guardrailChecker *GuardrailChecker,
	messageSender MessageSender,
	leadUpdater LeadUpdater,
	resetUC *ResetConversationUseCase,
	resetCommandEnabled bool,
	toolRegistry *ToolRegistry,
	toolResultMaxLength int,
	toolLoopMaxIter int,
	scoringFinder ScoringConfigFinder,
	handoffActivator HandoffActivator,
	crossSellRuleRepo specDomain.CrossSellRuleRepository,
	crossSellEvaluator *CrossSellRuleEvaluator,
	crossSellExecutor *CrossSellExecutor,
	log *zap.Logger,
) *ConversationEngine {
	return &ConversationEngine{
		providerRegistry:    providerRegistry,
		configResolver:      configResolver,
		stateRepo:           stateRepo,
		contextBuilder:      contextBuilder,
		stepEvaluator:       stepEvaluator,
		guardrailChecker:    guardrailChecker,
		messageSender:       messageSender,
		leadUpdater:         leadUpdater,
		resetUC:             resetUC,
		resetCommandEnabled: resetCommandEnabled,
		toolRegistry:        toolRegistry,
		toolResultMaxLength: toolResultMaxLength,
		toolLoopMaxIter:     toolLoopMaxIter,
		scoringFinder:       scoringFinder,
		handoffActivator:    handoffActivator,
		crossSellRuleRepo:   crossSellRuleRepo,
		crossSellEvaluator:  crossSellEvaluator,
		crossSellExecutor:   crossSellExecutor,
		turnLock:            newKeyedMutex(),
		log:                 log,
	}
}

// HandleMessages processes one or more debounced messages for a conversation.
func (e *ConversationEngine) HandleMessages(
	ctx context.Context,
	tenantID, conversationID, specialistID, productID string,
	messages []string,
) (err error) {
	ctx, span := observability.StartSpan(ctx, "ai.usecase.respond",
		attribute.String("tenant.id", tenantID),
		attribute.String("conversation.id", conversationID),
		attribute.String("specialist.id", specialistID),
	)
	defer span.End()

	// One turn at a time per conversation. A waiting turn re-reads state and
	// history after the holder commits, so it acts on the latest step instead
	// of clobbering it. Different conversations are unaffected.
	unlock := e.turnLock.Lock(conversationID)
	defer unlock()

	// Emit the cross-module latency histogram on every exit path so dashboards
	// can trend specialist responsiveness independently of the AI-module-local
	// aiRequestDuration (which is labeled by provider/model).
	reqStart := time.Now()
	defer func() {
		outcome := "ok"
		if err != nil {
			outcome = "error"
		}
		observability.SpecialistResponseDuration.WithLabelValues(outcome).Observe(time.Since(reqStart).Seconds())
	}()

	// 0. Intercept /reset command before any state loading.
	if e.resetCommandEnabled && e.resetUC != nil && IsResetCommand(messages) {
		return e.resetUC.Execute(ctx, tenantID, conversationID, "command")
	}

	// 1. Get or create ConversationState.
	state, err := e.stateRepo.FindByConversationID(ctx, conversationID)
	if err != nil {
		if err != domain.ErrConversationStateNotFound {
			return fmt.Errorf("conversation_engine: find state: %w", err)
		}
		// Create new state. A fresh UUID is assigned as the primary key so
		// GORM's Save on subsequent updates does an UPDATE (not a failing
		// INSERT on the empty-string PK).
		state, err = domain.NewConversationState(uuid.New().String(), conversationID, specialistID)
		if err != nil {
			return fmt.Errorf("conversation_engine: new state: %w", err)
		}
		if createErr := e.stateRepo.Create(ctx, state); createErr != nil {
			return fmt.Errorf("conversation_engine: create state: %w", createErr)
		}
	}

	// 1a. Self-heal the conversation's specialist. `specialistID` is the router's
	// fresh decision for this message; state.SpecialistID is the one persisted on
	// this conversation. They diverge legitimately after an explicit switch
	// (cross-sell/tool/automation) — those must stick. But the persisted specialist
	// stops being a valid choice when it is removed from the tenant OR deactivated in
	// admin; otherwise the conversation stays stuck on it forever. Heal in those
	// cases: adopt the router's specialist. Then treat state.SpecialistID as the
	// single source of truth downstream, keeping persona, config, scoring and
	// guardrails consistent within the turn.
	if state.SpecialistID != specialistID && !e.personaStillValid(ctx, state.SpecialistID, tenantID) {
		e.log.Info("conversation_engine: persisted specialist no longer valid (removed/inactive), re-routing conversation",
			zap.String("conversation_id", conversationID),
			zap.String("from_specialist_id", state.SpecialistID),
			zap.String("to_specialist_id", specialistID),
		)
		state.SpecialistID = specialistID
		if updateErr := e.stateRepo.Update(ctx, state); updateErr != nil {
			return fmt.Errorf("conversation_engine: persist healed specialist: %w", updateErr)
		}
	}
	specialistID = state.SpecialistID
	span.SetAttributes(attribute.String("specialist.id", specialistID))

	// 2. If handoff is active, skip AI processing.
	if state.HandoffActive {
		return nil
	}

	// 2a. Cross-sell: handle pending confirmation branch first (before any LLM call).
	if e.crossSellRuleRepo != nil && e.crossSellEvaluator != nil && e.crossSellExecutor != nil &&
		state.PendingCrossSellRuleID != nil {
		latestMsg := messages[len(messages)-1]
		if isAffirmative(latestMsg) {
			rule, findErr := e.crossSellRuleRepo.FindByID(ctx, *state.PendingCrossSellRuleID)
			if findErr != nil {
				return fmt.Errorf("conversation_engine: find pending cross-sell rule: %w", findErr)
			}
			if clearErr := e.crossSellExecutor.ClearPending(ctx, conversationID); clearErr != nil {
				e.log.Warn("conversation_engine: clear pending cross-sell failed",
					zap.String("conversation_id", conversationID),
					zap.Error(clearErr),
				)
			}
			crossSellColID := ""
			if e.scoringFinder != nil {
				if sc, scErr := e.scoringFinder.FindBySpecialistID(ctx, specialistID); scErr == nil && sc != nil {
					crossSellColID = sc.CrossSellColumnID
				}
			}
			originLeadID, leadIDErr := e.leadUpdater.GetLeadIDByConversation(ctx, conversationID)
			if leadIDErr != nil {
				return fmt.Errorf("conversation_engine: resolve origin lead id: %w", leadIDErr)
			}
			return e.crossSellExecutor.CompleteTransition(ctx, conversationID, tenantID, originLeadID, crossSellColID, rule)
		}
		// Negative answer: clear pending, fall through to normal flow.
		if clearErr := e.crossSellExecutor.ClearPending(ctx, conversationID); clearErr != nil {
			e.log.Warn("conversation_engine: clear pending cross-sell (negative) failed",
				zap.String("conversation_id", conversationID),
				zap.Error(clearErr),
			)
		}
		state.ClearPendingCrossSellRuleID()
	}

	// 2b. Cross-sell: evaluate active rules before invoking the LLM.
	if e.crossSellRuleRepo != nil && e.crossSellEvaluator != nil && e.crossSellExecutor != nil &&
		state.PendingCrossSellRuleID == nil {
		// Load specialist to check CrossSellEnabled flag.
		specialist, specErr := e.contextBuilder.SpecialistFinder.FindByID(ctx, specialistID)
		if specErr == nil && specialist != nil && specialist.CrossSellEnabled {
			activeRules, _ := e.crossSellRuleRepo.ListActiveBySpecialistOrdered(ctx, specialistID)
			if len(activeRules) > 0 {
				latestMsg := messages[len(messages)-1]
				match := e.crossSellEvaluator.Evaluate(activeRules, latestMsg, state.CollectedData)
				if match != nil {
					crossSellColID := ""
					if e.scoringFinder != nil {
						if sc, scErr2 := e.scoringFinder.FindBySpecialistID(ctx, specialistID); scErr2 == nil && sc != nil {
							crossSellColID = sc.CrossSellColumnID
						}
					}
					originLeadID, leadIDErr := e.leadUpdater.GetLeadIDByConversation(ctx, conversationID)
					if leadIDErr != nil {
						return fmt.Errorf("conversation_engine: resolve origin lead id: %w", leadIDErr)
					}
					return e.crossSellExecutor.Execute(ctx, conversationID, tenantID, originLeadID, crossSellColID, specialist, match)
				}
			}
		}
	}

	// 3. Resolve config.
	cfg := e.configResolver.Resolve(ctx, specialistID)
	span.SetAttributes(
		attribute.String("ai.provider", cfg.Provider),
		attribute.String("ai.model", cfg.Model),
	)

	// 4. Build AI request context.
	req, err := e.contextBuilder.Build(ctx, state, productID, 20)
	if err != nil {
		return fmt.Errorf("conversation_engine: build context: %w", err)
	}

	// 5. Set provider/model/temperature/maxTokens from config.
	req.Provider = cfg.Provider
	req.Model = cfg.Model
	req.Temperature = cfg.Temperature
	req.MaxTokens = cfg.MaxTokens

	// 6. Get provider and generate response.
	provider, err := e.providerRegistry.Get(cfg.Provider)
	if err != nil {
		return fmt.Errorf("conversation_engine: get provider: %w", err)
	}

	start := time.Now()
	resp, err := e.executeToolLoop(ctx, provider, req, tenantID, specialistID, e.toolLoopMaxIter, e.toolResultMaxLength)
	elapsed := time.Since(start).Seconds()

	status := "success"
	if err != nil {
		status = "error"
		aiRequestsTotal.WithLabelValues(tenantID, specialistID, cfg.Provider, cfg.Model, status).Inc()
		// Never leave the client staring at silence. The turn still fails (logs,
		// metrics and the caller's error path are unchanged) — this only makes
		// the failure visible on WhatsApp, where a mute specialist reads as a
		// broken product. State is deliberately not advanced.
		if sendErr := e.messageSender.SendAIResponse(ctx, tenantID, conversationID, aiUnavailableFallbackMessage); sendErr != nil {
			e.log.Warn("conversation_engine: fallback message send failed",
				zap.String("conversation_id", conversationID),
				zap.Error(sendErr),
			)
		}
		return fmt.Errorf("conversation_engine: generate response: %w", err)
	}

	// 7. Record metrics.
	aiRequestsTotal.WithLabelValues(tenantID, specialistID, cfg.Provider, cfg.Model, status).Inc()
	aiRequestDuration.WithLabelValues(tenantID, cfg.Provider, cfg.Model).Observe(elapsed)
	aiTokensPromptTotal.WithLabelValues(tenantID, specialistID, cfg.Provider).Add(float64(resp.PromptTokens))
	aiTokensCompletionTotal.WithLabelValues(tenantID, specialistID, cfg.Provider).Add(float64(resp.CompletionTokens))

	// 8. Parse metadata from response.
	meta, cleanedContent := e.stepEvaluator.ParseMetadata(resp.Content)

	// 9. Evaluate steps.
	steps, _ := e.contextBuilder.StepFinder.FindBySpecialistID(ctx, specialistID)
	combinedMessage := strings.Join(messages, " ")

	var stepCompleted bool
	var collectedData string
	var stepScore int
	var targetColumnID string

	if len(steps) > 0 && state.CurrentStepIndex < len(steps) {
		currentStep := steps[state.CurrentStepIndex]

		// Try rule-based evaluation first.
		evalResult := e.stepEvaluator.EvaluateByRule(&currentStep, combinedMessage)
		if evalResult.Completed {
			stepCompleted = true
			collectedData = evalResult.CollectedData
			stepScore = currentStep.Score
			targetColumnID = currentStep.TargetColumnID
		} else if meta.StepCompleted {
			// Fallback to LLM metadata.
			stepCompleted = true
			collectedData = meta.CollectedData
			stepScore = meta.Score
			targetColumnID = currentStep.TargetColumnID
		}

		// 10. If step completed → advance step, update lead score, move to column.
		if stepCompleted {
			state.AdvanceStep(fmt.Sprintf("step_%d", state.CurrentStepIndex), collectedData, stepScore)
			aiStepsCompletedTotal.WithLabelValues(tenantID, specialistID).Inc()

			if e.leadUpdater != nil {
				if updateErr := e.leadUpdater.UpdateLeadScore(ctx, conversationID, state.AccumulatedScore); updateErr != nil {
					e.log.Warn("conversation_engine: update lead score failed",
						zap.String("conversation_id", conversationID),
						zap.Error(updateErr),
					)
				}

				// Resolve scoring config once; reused below for LLM veto,
				// early qualification, and end-of-flow disqualification.
				var sc *specDomain.ScoringConfig
				if e.scoringFinder != nil {
					loaded, scErr := e.scoringFinder.FindBySpecialistID(ctx, specialistID)
					if scErr == nil {
						sc = loaded
					} else if !errors.Is(scErr, specDomain.ErrScoringConfigNotFound) {
						e.log.Warn("conversation_engine: load scoring config failed",
							zap.String("specialist_id", specialistID),
							zap.Error(scErr),
						)
					}
				}

				// Column routing priority:
				//   1. LLM veto flag (meta.Disqualified) overrides everything.
				//   2. Explicit step.TargetColumnID for forward progression.
				//   3. OutcomeCalculator: Aprovado → qualified, Humano → human + handoff, Reprovado → disqualified.
				effectiveTarget := ""
				disqualifying := false
				switch {
				case meta.Disqualified && sc != nil && sc.DisqualifiedColumnID != "":
					effectiveTarget = sc.DisqualifiedColumnID
					disqualifying = true
				case targetColumnID != "":
					effectiveTarget = targetColumnID
				case sc != nil:
					outcome := specDomain.CalculateOutcome(sc, state.AccumulatedScore)
					QualificationOutcomeTotal.WithLabelValues(tenantID, specialistID, string(outcome)).Inc()
					if setErr := e.leadUpdater.SetOutcome(ctx, conversationID, string(outcome)); setErr != nil {
						e.log.Warn("conversation_engine: set outcome failed",
							zap.String("conversation_id", conversationID),
							zap.String("outcome", string(outcome)),
							zap.Error(setErr),
						)
					}
					switch outcome {
					case specDomain.OutcomeAprovado:
						if sc.QualifiedColumnID != "" {
							effectiveTarget = sc.QualifiedColumnID
						}
					case specDomain.OutcomeHumano:
						if sc.HumanColumnID != "" {
							effectiveTarget = sc.HumanColumnID
						}
						if e.handoffActivator != nil {
							if hErr := e.handoffActivator.Activate(ctx, conversationID); hErr != nil {
								e.log.Warn("conversation_engine: handoff activation failed",
									zap.String("conversation_id", conversationID),
									zap.Error(hErr),
								)
							}
						}
					case specDomain.OutcomeReprovado:
						if state.CurrentStepIndex >= len(steps) && sc.DisqualifiedColumnID != "" {
							effectiveTarget = sc.DisqualifiedColumnID
							disqualifying = true
						}
					}
				}

				if effectiveTarget != "" {
					if moveErr := e.leadUpdater.MoveLeadToColumn(ctx, conversationID, effectiveTarget); moveErr != nil {
						e.log.Warn("conversation_engine: move lead to column failed",
							zap.String("conversation_id", conversationID),
							zap.String("column_id", effectiveTarget),
							zap.Error(moveErr),
						)
					} else if disqualifying {
						aiLeadsDisqualifiedTotal.WithLabelValues(tenantID, specialistID).Inc()
					}
				}
			}

			// 11. If all steps completed → log qualification.
			if state.CurrentStepIndex >= len(steps) {
				aiLeadsQualifiedTotal.WithLabelValues(tenantID, specialistID).Inc()
				e.log.Info("conversation_engine: all steps completed, lead qualified",
					zap.String("tenant_id", tenantID),
					zap.String("conversation_id", conversationID),
					zap.String("specialist_id", specialistID),
					zap.Int("accumulated_score", state.AccumulatedScore),
				)
			}
		}
	}

	// 12. Check guardrails.
	responseContent := cleanedContent
	guardrails, _ := e.contextBuilder.GuardrailFinder.FindBySpecialistID(ctx, specialistID)
	if violated, fallback := e.guardrailChecker.Check(responseContent, guardrails); violated {
		aiGuardrailViolationsTotal.WithLabelValues(tenantID, specialistID, string(specDomain.GuardrailTypeForbiddenTopics)).Inc()
		e.log.Warn("conversation_engine: guardrail violated, using fallback message",
			zap.String("tenant_id", tenantID),
			zap.String("conversation_id", conversationID),
		)
		responseContent = fallback
	}

	// 13. Persist state BEFORE sending. If we send first and the send fails, the
	// in-memory state.AdvanceStep changes get lost while lead score/column (which
	// were already persisted above) stay advanced — state and lead diverge and
	// the next message re-completes the same step with a double score.
	if updateErr := e.stateRepo.Update(ctx, state); updateErr != nil {
		return fmt.Errorf("conversation_engine: update state: %w", updateErr)
	}

	// 14. Send AI response.
	if sendErr := e.messageSender.SendAIResponse(ctx, tenantID, conversationID, responseContent); sendErr != nil {
		return fmt.Errorf("conversation_engine: send response: %w", sendErr)
	}

	return nil
}

// isAffirmative returns true when s is a recognisable affirmative reply (accent-insensitive).
func isAffirmative(s string) bool {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, strings.ToLower(strings.TrimSpace(s)))
	return affirmativeRe.MatchString(normalized)
}

var affirmativeRe = regexp.MustCompile(`^(sim|s|ok|claro|pode|por favor)\b`)

// executeToolLoop runs the tool calling loop: send request → get tool calls → execute → repeat
// until the response has no tool calls or maxIterations is reached.
// If toolRegistry is nil, it falls back to a single direct provider call (no tool loop).
func (e *ConversationEngine) executeToolLoop(
	ctx context.Context,
	provider domain.AIProvider,
	req *domain.AIRequest,
	tenantID, specialistID string,
	maxIterations int,
	resultMaxLength int,
) (*domain.AIResponse, error) {
	if e.toolRegistry == nil {
		return provider.GenerateResponse(ctx, req)
	}

	for i := 0; i < maxIterations; i++ {
		resp, err := provider.GenerateResponse(ctx, req)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			aiToolLoopIterations.WithLabelValues(tenantID, specialistID).Observe(float64(i + 1))
			return resp, nil
		}

		var results []domain.ToolResult
		for _, call := range resp.ToolCalls {
			tool, tErr := e.toolRegistry.Get(call.ToolName)
			if tErr != nil {
				aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "not_found").Inc()
				results = append(results, domain.NewToolResult(call.ID, "tool not found: "+call.ToolName, true))
				continue
			}

			start := time.Now()
			result, execErr := tool.Execute(ctx, tenantID, call.Arguments)
			elapsed := time.Since(start)
			aiToolCallDurationSeconds.WithLabelValues(tenantID, call.ToolName).Observe(elapsed.Seconds())

			if execErr != nil {
				aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "error").Inc()
				e.log.Warn("tool_call_failed",
					zap.String("tenant_id", tenantID),
					zap.String("tool_name", call.ToolName),
					zap.Error(execErr),
				)
				results = append(results, domain.NewToolResult(call.ID, "error: "+execErr.Error(), true))
				continue
			}

			// Truncate if needed.
			if resultMaxLength > 0 && len(result.Content) > resultMaxLength {
				aiToolResultTruncatedTotal.WithLabelValues(tenantID, call.ToolName).Inc()
				result.Content = result.Content[:resultMaxLength]
			}

			aiToolCallsTotal.WithLabelValues(tenantID, specialistID, call.ToolName, "success").Inc()
			e.log.Info("tool_call_executed",
				zap.String("tenant_id", tenantID),
				zap.String("specialist_id", specialistID),
				zap.String("tool_name", call.ToolName),
				zap.Duration("duration", elapsed),
			)
			results = append(results, *result)
		}

		// Backfill ToolCallID on results that didn't set it (e.g. Execute returned result with empty ID).
		for idx := range results {
			if results[idx].ToolCallID == "" && idx < len(resp.ToolCalls) {
				results[idx].ToolCallID = resp.ToolCalls[idx].ID
			}
		}

		req.ToolResults = results
	}

	e.log.Warn("tool_loop_max_iterations",
		zap.String("tenant_id", tenantID),
		zap.Int("max_iterations", maxIterations),
	)
	return nil, fmt.Errorf("tool loop exceeded max iterations (%d)", maxIterations)
}
