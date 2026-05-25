package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// --- in-memory fakes (no DB) for the notes adapter unit test ---

type notesFakeLeadRepo struct {
	leads    map[string]*domain.Lead // by ID
	forceErr error                   // when set, FindCurrentByConversationID returns it
}

func (r *notesFakeLeadRepo) Create(_ context.Context, l *domain.Lead) error {
	r.leads[l.ID] = l
	return nil
}
func (r *notesFakeLeadRepo) FindByID(_ context.Context, id string) (*domain.Lead, error) {
	if l, ok := r.leads[id]; ok {
		return l, nil
	}
	return nil, domain.ErrLeadNotFound
}
func (r *notesFakeLeadRepo) Update(_ context.Context, l *domain.Lead) error {
	r.leads[l.ID] = l
	return nil
}
func (r *notesFakeLeadRepo) FindByContactAndTenant(_ context.Context, _, _ string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}
func (r *notesFakeLeadRepo) FindByConversationID(_ context.Context, _ string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}
func (r *notesFakeLeadRepo) FindCurrentByConversationID(_ context.Context, tenantID, conversationID string) (*domain.Lead, error) {
	if r.forceErr != nil {
		return nil, r.forceErr
	}
	for _, l := range r.leads {
		if l.ConversationID == conversationID && l.TenantID == tenantID {
			return l, nil
		}
	}
	return nil, domain.ErrLeadNotFound
}
func (r *notesFakeLeadRepo) FindByFunnelID(_ context.Context, _ string, _ domain.LeadFilter) (*domain.LeadList, error) {
	return &domain.LeadList{}, nil
}
func (r *notesFakeLeadRepo) CountByColumnID(_ context.Context, _ string) (int, error) { return 0, nil }
func (r *notesFakeLeadRepo) FindByTenantAndSearch(_ context.Context, _, _ string, _ int) ([]domain.Lead, error) {
	return nil, nil
}

type notesFakeNoteRepo struct {
	notes []domain.LeadNote
}

func (r *notesFakeNoteRepo) Create(_ context.Context, n *domain.LeadNote) error {
	r.notes = append(r.notes, *n)
	return nil
}
func (r *notesFakeNoteRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadNote, error) {
	var out []domain.LeadNote
	for _, n := range r.notes {
		if n.LeadID == leadID {
			out = append(out, n)
		}
	}
	return out, nil
}

type notesFakeUserNames struct{ name string }

func (s notesFakeUserNames) FindNameByID(_ context.Context, _ string) (string, error) {
	return s.name, nil
}

func newNotesAdapterTest() (*WhatsAppNotesAdapter, *notesFakeLeadRepo) {
	leadRepo := &notesFakeLeadRepo{leads: map[string]*domain.Lead{}}
	noteRepo := &notesFakeNoteRepo{}
	createUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)
	a := NewWhatsAppNotesAdapter(leadRepo, noteRepo, notesFakeUserNames{name: "Maria"}, createUC)
	return a, leadRepo
}

func TestNotesAdapter_AddAndList(t *testing.T) {
	ctx := context.Background()
	a, leadRepo := newNotesAdapterTest()

	lead, err := domain.NewLead(uuid.New().String(), "tenant-1", "f1", "c1", "contact-1", "conv-1")
	require.NoError(t, err)
	require.NoError(t, leadRepo.Create(ctx, lead))

	notes, err := a.AddNote(ctx, "tenant-1", "conv-1", "primeira nota", uuid.New().String())
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "primeira nota", notes[0].Content)
	assert.Equal(t, "Maria", notes[0].AuthorName)

	has, list, err := a.NotesForConversation(ctx, "tenant-1", "conv-1")
	require.NoError(t, err)
	assert.True(t, has)
	require.Len(t, list, 1)
	assert.Equal(t, "primeira nota", list[0].Content)
}

func TestNotesAdapter_NoLead(t *testing.T) {
	a, _ := newNotesAdapterTest()
	has, list, err := a.NotesForConversation(context.Background(), "tenant-1", "conv-x")
	require.NoError(t, err)
	assert.False(t, has)
	assert.Empty(t, list)
}

func TestNotesAdapter_AddNote_NoLead(t *testing.T) {
	a, _ := newNotesAdapterTest()
	_, err := a.AddNote(context.Background(), "tenant-1", "conv-x", "x", "user-1")
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

func TestNotesAdapter_AddNote_EmptyContentRejected(t *testing.T) {
	ctx := context.Background()
	a, leadRepo := newNotesAdapterTest()

	lead, err := domain.NewLead(uuid.New().String(), "tenant-1", "f1", "c1", "contact-1", "conv-1")
	require.NoError(t, err)
	require.NoError(t, leadRepo.Create(ctx, lead))

	_, err = a.AddNote(ctx, "tenant-1", "conv-1", "", "user-1")
	assert.ErrorIs(t, err, domain.ErrNoteContentRequired)
}

func TestNotesAdapter_GenericLookupErrorPropagates(t *testing.T) {
	a, leadRepo := newNotesAdapterTest()
	leadRepo.forceErr = errors.New("db down")

	_, _, err := a.NotesForConversation(context.Background(), "tenant-1", "conv-1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrLeadNotFound)
}
