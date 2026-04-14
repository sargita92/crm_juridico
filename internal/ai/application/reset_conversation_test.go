package application

import (
	"context"
	"errors"
	"testing"

	aidomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubStateRepo struct {
	state      *aidomain.ConversationState
	updateErr  error
	updateCall int
}

func (s *stubStateRepo) Create(_ context.Context, _ *aidomain.ConversationState) error {
	return nil
}
func (s *stubStateRepo) FindByConversationID(_ context.Context, _ string) (*aidomain.ConversationState, error) {
	if s.state == nil {
		return nil, aidomain.ErrConversationStateNotFound
	}
	return s.state, nil
}
func (s *stubStateRepo) Update(_ context.Context, st *aidomain.ConversationState) error {
	s.updateCall++
	s.state = st
	return s.updateErr
}

type stubEntryColumn struct {
	id  string
	err error
}

func (s stubEntryColumn) FindEntryColumnID(_ context.Context, _ string) (string, error) {
	return s.id, s.err
}

type stubLeadResetter struct {
	lastConvID string
	lastColID  string
	err        error
}

func (s *stubLeadResetter) ResetLeadForConversation(_ context.Context, convID, colID string) error {
	s.lastConvID, s.lastColID = convID, colID
	return s.err
}

type stubSender struct {
	lastContent string
	err         error
}

func (s *stubSender) SendAIResponse(_ context.Context, _, _, content string) error {
	s.lastContent = content
	return s.err
}

func TestResetConversation_ZeroesStateAndMovesLead(t *testing.T) {
	state, err := aidomain.NewConversationState("id-1", "conv-1", "spec-1")
	require.NoError(t, err)
	state.AdvanceStep("nome", "Maria", 10)

	stateRepo := &stubStateRepo{state: state}
	entry := stubEntryColumn{id: "col-entry"}
	resetter := &stubLeadResetter{}
	sender := &stubSender{}

	uc := NewResetConversationUseCase(stateRepo, entry, resetter, sender, zap.NewNop())

	err = uc.Execute(context.Background(), "tenant-1", "conv-1", "command")
	require.NoError(t, err)

	assert.Equal(t, 0, stateRepo.state.CurrentStepIndex)
	assert.Equal(t, 0, stateRepo.state.AccumulatedScore)
	assert.Equal(t, 1, stateRepo.updateCall)
	assert.Equal(t, "conv-1", resetter.lastConvID)
	assert.Equal(t, "col-entry", resetter.lastColID)
	assert.Contains(t, sender.lastContent, "reiniciada")
}

func TestResetConversation_NoStateYet_StillSendsConfirmation(t *testing.T) {
	stateRepo := &stubStateRepo{state: nil}
	sender := &stubSender{}
	uc := NewResetConversationUseCase(stateRepo, stubEntryColumn{id: "col-entry"}, &stubLeadResetter{}, sender, zap.NewNop())
	err := uc.Execute(context.Background(), "tenant-1", "conv-1", "command")
	require.NoError(t, err)
	assert.Contains(t, sender.lastContent, "reiniciada")
}

func TestResetConversation_NoEntryColumn_SkipsLeadMove(t *testing.T) {
	state, _ := aidomain.NewConversationState("id-1", "conv-1", "spec-1")
	state.AdvanceStep("x", "y", 5)
	stateRepo := &stubStateRepo{state: state}
	resetter := &stubLeadResetter{}
	uc := NewResetConversationUseCase(stateRepo, stubEntryColumn{id: ""}, resetter, &stubSender{}, zap.NewNop())
	err := uc.Execute(context.Background(), "tenant-1", "conv-1", "command")
	require.NoError(t, err)
	assert.Equal(t, "", resetter.lastColID, "lead reset should be skipped when no entry column")
}

func TestResetConversation_UpdateErrorPropagated(t *testing.T) {
	state, _ := aidomain.NewConversationState("id-1", "conv-1", "spec-1")
	stateRepo := &stubStateRepo{state: state, updateErr: errors.New("boom")}
	uc := NewResetConversationUseCase(stateRepo, stubEntryColumn{id: "col-entry"}, &stubLeadResetter{}, &stubSender{}, zap.NewNop())
	err := uc.Execute(context.Background(), "tenant-1", "conv-1", "command")
	assert.Error(t, err)
}
