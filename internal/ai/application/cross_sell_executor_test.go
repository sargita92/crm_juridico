package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/ai/application"
	specDomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

type fakeProductSpecialistResolver struct {
	specialistID    string
	funnelID        string
	initialColumnID string
	err             error
}

func (f *fakeProductSpecialistResolver) FindSpecialistByProduct(_ context.Context, _ string) (string, string, string, error) {
	return f.specialistID, f.funnelID, f.initialColumnID, f.err
}

type fakeLeadFactory struct {
	capturedOriginLeadID string
	capturedSpecialistID string
	returnedLeadID       string
	err                  error
}

func (f *fakeLeadFactory) CreateForCrossSell(_ context.Context, originLeadID, _, _, _, specialistID string) (string, error) {
	f.capturedOriginLeadID = originLeadID
	f.capturedSpecialistID = specialistID
	return f.returnedLeadID, f.err
}

type fakeConversationMover struct {
	migratedSpecialistID  string
	pendingRuleID         string
	clearCalled           bool
	migrateErr            error
	setPendingErr         error
	clearErr              error
}

func (f *fakeConversationMover) MigrateSpecialist(_ context.Context, _, newSpecialistID string) error {
	f.migratedSpecialistID = newSpecialistID
	return f.migrateErr
}

func (f *fakeConversationMover) SetPendingCrossSell(_ context.Context, _, ruleID string) error {
	f.pendingRuleID = ruleID
	return f.setPendingErr
}

func (f *fakeConversationMover) ClearPendingCrossSell(_ context.Context, _ string) error {
	f.clearCalled = true
	return f.clearErr
}

type fakeLeadUpdater struct {
	outcomeSet    string
	movedColumnID string
	setOutcomeErr error
	moveErr       error
}

func (f *fakeLeadUpdater) UpdateLeadScore(_ context.Context, _ string, _ int) error { return nil }

func (f *fakeLeadUpdater) MoveLeadToColumn(_ context.Context, _, columnID string) error {
	f.movedColumnID = columnID
	return f.moveErr
}

func (f *fakeLeadUpdater) SetOutcome(_ context.Context, _ string, outcome string) error {
	f.outcomeSet = outcome
	return f.setOutcomeErr
}

type fakeMessageSender struct {
	sentMessages []string
	err          error
}

func (f *fakeMessageSender) SendAIResponse(_ context.Context, _, _, content string) error {
	f.sentMessages = append(f.sentMessages, content)
	return f.err
}

type fakeProductNameLookup struct {
	name string
	err  error
}

func (f *fakeProductNameLookup) Name(_ context.Context, _ string) (string, error) {
	return f.name, f.err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestExecutor(
	resolver *fakeProductSpecialistResolver,
	factory *fakeLeadFactory,
	mover *fakeConversationMover,
	updater *fakeLeadUpdater,
	sender *fakeMessageSender,
	lookup *fakeProductNameLookup,
) *application.CrossSellExecutor {
	return application.NewCrossSellExecutor(resolver, factory, mover, updater, sender, lookup)
}

func makeRule(productID string) *specDomain.CrossSellRule {
	r, err := specDomain.NewCrossSellRule(
		"rule-1", "spec-origin", 0,
		specDomain.CrossSellTriggerKeyword,
		specDomain.KeywordTrigger{Termos: []string{"imóvel"}},
		productID,
	)
	if err != nil {
		panic(err)
	}
	return r
}

func makeSpecialist(mode specDomain.CrossSellMode, tmpl string) *specDomain.Specialist {
	s, err := specDomain.NewSpecialist("spec-origin", "Origem", "", "prompt origin")
	if err != nil {
		panic(err)
	}
	if err := s.EnableCrossSell(mode, tmpl); err != nil {
		panic(err)
	}
	return s
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestCrossSell_AnnounceMode verifies that:
//   - MessageSender is called with the rendered template ({{produto}} replaced).
//   - CompleteTransition fires: lead created, outcome set, lead moved, specialist migrated.
func TestCrossSell_AnnounceMode(t *testing.T) {
	resolver := &fakeProductSpecialistResolver{specialistID: "spec-new", funnelID: "funnel-1", initialColumnID: "col-new"}
	factory := &fakeLeadFactory{returnedLeadID: "lead-new"}
	mover := &fakeConversationMover{}
	updater := &fakeLeadUpdater{}
	sender := &fakeMessageSender{}
	lookup := &fakeProductNameLookup{name: "Imóveis"}

	spec := makeSpecialist(specDomain.CrossSellModeAnnounce, "Conheça nosso especialista em {{produto}}!")
	rule := makeRule("prod-imovel")

	exec := newTestExecutor(resolver, factory, mover, updater, sender, lookup)
	err := exec.Execute(context.Background(), "conv-1", "tenant-1", "lead-origin", "col-cs", spec, rule)
	require.NoError(t, err)

	// message sent with placeholder replaced
	require.Len(t, sender.sentMessages, 1)
	assert.Equal(t, "Conheça nosso especialista em Imóveis!", sender.sentMessages[0])

	// transition completed
	assert.Equal(t, "lead-origin", factory.capturedOriginLeadID)
	assert.Equal(t, "spec-new", factory.capturedSpecialistID)
	assert.Equal(t, string(specDomain.OutcomeCrossSell), updater.outcomeSet)
	assert.Equal(t, "col-cs", updater.movedColumnID)
	assert.Equal(t, "spec-new", mover.migratedSpecialistID)

	// confirm pending must NOT be set in announce mode
	assert.Empty(t, mover.pendingRuleID)
}

// TestCrossSell_SilentMode verifies that no message is sent but transition completes.
func TestCrossSell_SilentMode(t *testing.T) {
	resolver := &fakeProductSpecialistResolver{specialistID: "spec-new", funnelID: "funnel-1", initialColumnID: "col-new"}
	factory := &fakeLeadFactory{returnedLeadID: "lead-new"}
	mover := &fakeConversationMover{}
	updater := &fakeLeadUpdater{}
	sender := &fakeMessageSender{}
	lookup := &fakeProductNameLookup{name: "Trabalhista"}

	spec := makeSpecialist(specDomain.CrossSellModeSilent, "")
	rule := makeRule("prod-trabalhista")

	exec := newTestExecutor(resolver, factory, mover, updater, sender, lookup)
	err := exec.Execute(context.Background(), "conv-2", "tenant-1", "lead-origin", "", spec, rule)
	require.NoError(t, err)

	// no message
	assert.Empty(t, sender.sentMessages)

	// transition still completed
	assert.Equal(t, "lead-origin", factory.capturedOriginLeadID)
	assert.Equal(t, string(specDomain.OutcomeCrossSell), updater.outcomeSet)
	assert.Equal(t, "spec-new", mover.migratedSpecialistID)

	// no column move when crossSellColumnID is empty
	assert.Empty(t, updater.movedColumnID)
}

// TestCrossSell_ConfirmMode verifies that:
//   - MessageSender is called with the confirmation question.
//   - SetPendingCrossSell is called with the rule ID.
//   - NO transition happens (no lead created, no outcome set, no specialist migrated).
func TestCrossSell_ConfirmMode(t *testing.T) {
	resolver := &fakeProductSpecialistResolver{specialistID: "spec-new", funnelID: "funnel-1", initialColumnID: "col-new"}
	factory := &fakeLeadFactory{}
	mover := &fakeConversationMover{}
	updater := &fakeLeadUpdater{}
	sender := &fakeMessageSender{}
	lookup := &fakeProductNameLookup{name: "Previdenciário"}

	spec := makeSpecialist(specDomain.CrossSellModeConfirm, "")
	rule := makeRule("prod-prev")

	exec := newTestExecutor(resolver, factory, mover, updater, sender, lookup)
	err := exec.Execute(context.Background(), "conv-3", "tenant-1", "lead-origin", "col-cs", spec, rule)
	require.NoError(t, err)

	// confirmation question sent
	require.Len(t, sender.sentMessages, 1)
	assert.Contains(t, sender.sentMessages[0], "Previdenciário")

	// pending cross-sell marked
	assert.Equal(t, rule.ID, mover.pendingRuleID)

	// NO transition
	assert.Empty(t, factory.capturedOriginLeadID)
	assert.Empty(t, updater.outcomeSet)
	assert.Empty(t, mover.migratedSpecialistID)
}

// TestCrossSell_OriginLeadIDPassedToFactory verifies the origin lead ID is forwarded.
func TestCrossSell_OriginLeadIDPassedToFactory(t *testing.T) {
	resolver := &fakeProductSpecialistResolver{specialistID: "spec-new", funnelID: "f1", initialColumnID: "c1"}
	factory := &fakeLeadFactory{returnedLeadID: "lead-xyz"}
	mover := &fakeConversationMover{}
	updater := &fakeLeadUpdater{}
	sender := &fakeMessageSender{}
	lookup := &fakeProductNameLookup{name: "Família"}

	spec := makeSpecialist(specDomain.CrossSellModeSilent, "")
	rule := makeRule("prod-familia")

	exec := newTestExecutor(resolver, factory, mover, updater, sender, lookup)
	err := exec.Execute(context.Background(), "conv-4", "tenant-1", "lead-abc123", "", spec, rule)
	require.NoError(t, err)

	assert.Equal(t, "lead-abc123", factory.capturedOriginLeadID)
}

// TestCrossSell_ProductNameLookupError propagates error from lookup.
func TestCrossSell_ProductNameLookupError(t *testing.T) {
	lookup := &fakeProductNameLookup{err: errors.New("lookup failed")}
	spec := makeSpecialist(specDomain.CrossSellModeSilent, "")
	rule := makeRule("prod-x")

	exec := newTestExecutor(
		&fakeProductSpecialistResolver{},
		&fakeLeadFactory{},
		&fakeConversationMover{},
		&fakeLeadUpdater{},
		&fakeMessageSender{},
		lookup,
	)
	err := exec.Execute(context.Background(), "conv-5", "tenant-1", "lead-1", "", spec, rule)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup failed")
}

// TestCrossSell_CompleteTransition_DirectCall verifies CompleteTransition can be called
// independently (as the confirm handler would do after a positive answer).
func TestCrossSell_CompleteTransition_DirectCall(t *testing.T) {
	resolver := &fakeProductSpecialistResolver{specialistID: "spec-new", funnelID: "f1", initialColumnID: "c1"}
	factory := &fakeLeadFactory{returnedLeadID: "lead-new"}
	mover := &fakeConversationMover{}
	updater := &fakeLeadUpdater{}
	sender := &fakeMessageSender{}
	lookup := &fakeProductNameLookup{name: "Imóveis"}

	rule := makeRule("prod-imovel")

	exec := newTestExecutor(resolver, factory, mover, updater, sender, lookup)
	err := exec.CompleteTransition(context.Background(), "conv-6", "tenant-1", "lead-origin", "col-cs", rule)
	require.NoError(t, err)

	// no message sent by CompleteTransition itself
	assert.Empty(t, sender.sentMessages)

	assert.Equal(t, "lead-origin", factory.capturedOriginLeadID)
	assert.Equal(t, string(specDomain.OutcomeCrossSell), updater.outcomeSet)
	assert.Equal(t, "col-cs", updater.movedColumnID)
	assert.Equal(t, "spec-new", mover.migratedSpecialistID)
}
