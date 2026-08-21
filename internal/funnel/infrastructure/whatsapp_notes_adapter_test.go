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
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
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
func (r *notesFakeLeadRepo) FindByContactAndTenant(_ context.Context, tenantID, contactID string) (*domain.Lead, error) {
	for _, l := range r.leads {
		if l.ContactID == contactID && l.TenantID == tenantID {
			return l, nil
		}
	}
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

// notesFakeConversations resolves conversation -> contact, like the whatsapp repo.
type notesFakeConversations struct {
	convs map[string]*whatsappdomain.Conversation
}

func (r *notesFakeConversations) FindByID(_ context.Context, id string) (*whatsappdomain.Conversation, error) {
	if c, ok := r.convs[id]; ok {
		return c, nil
	}
	return nil, whatsappdomain.ErrConversationNotFound
}

// notesFakeLeadCreator stands in for CreateLeadUseCase: it materialises the lead
// the real one would create, or fails when configured to.
type notesFakeLeadCreator struct {
	leadRepo *notesFakeLeadRepo
	err      error
	calls    int
}

func (c *notesFakeLeadCreator) CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID, _ string) error {
	c.calls++
	if c.err != nil {
		return c.err
	}
	lead, err := domain.NewLead(uuid.New().String(), tenantID, "f1", "c1", contactID, conversationID)
	if err != nil {
		return err
	}
	return c.leadRepo.Create(ctx, lead)
}

type notesAdapterFixture struct {
	adapter  *WhatsAppNotesAdapter
	leadRepo *notesFakeLeadRepo
	convs    *notesFakeConversations
	creator  *notesFakeLeadCreator
}

func newNotesAdapterTest() *notesAdapterFixture {
	leadRepo := &notesFakeLeadRepo{leads: map[string]*domain.Lead{}}
	noteRepo := &notesFakeNoteRepo{}
	createUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)
	convs := &notesFakeConversations{convs: map[string]*whatsappdomain.Conversation{}}
	creator := &notesFakeLeadCreator{leadRepo: leadRepo}
	a := NewWhatsAppNotesAdapter(leadRepo, noteRepo, notesFakeUserNames{name: "Maria"}, createUC, convs, creator)
	return &notesAdapterFixture{adapter: a, leadRepo: leadRepo, convs: convs, creator: creator}
}

// addConversation registers a conversation owned by tenantID.
func (f *notesAdapterFixture) addConversation(t *testing.T, tenantID, convID, contactID string) {
	t.Helper()
	conv, err := whatsappdomain.NewConversation(convID, tenantID, contactID)
	require.NoError(t, err)
	f.convs.convs[convID] = conv
}

func TestNotesAdapter_AddAndList(t *testing.T) {
	ctx := context.Background()
	f := newNotesAdapterTest()

	lead, err := domain.NewLead(uuid.New().String(), "tenant-1", "f1", "c1", "contact-1", "conv-1")
	require.NoError(t, err)
	require.NoError(t, f.leadRepo.Create(ctx, lead))

	notes, err := f.adapter.AddNote(ctx, "tenant-1", "conv-1", "primeira nota", uuid.New().String())
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "primeira nota", notes[0].Content)
	assert.Equal(t, "Maria", notes[0].AuthorName)
	assert.Zero(t, f.creator.calls, "lead already existed, nothing to promote")

	list, err := f.adapter.NotesForConversation(ctx, "tenant-1", "conv-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "primeira nota", list[0].Content)
}

// Rendering the panel must never mutate the funnel: a lead-less conversation
// reports no notes and stays lead-less.
func TestNotesAdapter_NoLead_ListsNothingWithoutCreating(t *testing.T) {
	f := newNotesAdapterTest()
	f.addConversation(t, "tenant-1", "conv-x", "contact-9")

	list, err := f.adapter.NotesForConversation(context.Background(), "tenant-1", "conv-x")
	require.NoError(t, err)
	assert.Empty(t, list)
	assert.Zero(t, f.creator.calls, "listing must not create a lead")
}

// The chat always offers the note field, so writing the first note on a
// conversation that never became a lead has to work — it promotes the
// conversation instead of failing.
func TestNotesAdapter_AddNote_PromotesConversationWithoutLead(t *testing.T) {
	ctx := context.Background()
	f := newNotesAdapterTest()
	f.addConversation(t, "tenant-1", "conv-x", "contact-9")

	notes, err := f.adapter.AddNote(ctx, "tenant-1", "conv-x", "cliente pediu retorno", "user-1")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "cliente pediu retorno", notes[0].Content)
	assert.Equal(t, 1, f.creator.calls)

	// And the note is readable back through the normal listing path.
	list, err := f.adapter.NotesForConversation(ctx, "tenant-1", "conv-x")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

// CreateLead deduplicates by contact and never re-points an existing lead at a
// new conversation, so a contact can hold a lead created on an earlier one.
// Reads and writes must land on that same lead, otherwise the note is saved
// somewhere the panel does not read back from.
func TestNotesAdapter_FallsBackToContactLeadFromAnotherConversation(t *testing.T) {
	ctx := context.Background()
	f := newNotesAdapterTest()
	f.addConversation(t, "tenant-1", "conv-new", "contact-1")

	lead, err := domain.NewLead(uuid.New().String(), "tenant-1", "f1", "c1", "contact-1", "conv-old")
	require.NoError(t, err)
	require.NoError(t, f.leadRepo.Create(ctx, lead))

	notes, err := f.adapter.AddNote(ctx, "tenant-1", "conv-new", "nota", "user-1")
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Zero(t, f.creator.calls, "the contact already had a lead")

	list, err := f.adapter.NotesForConversation(ctx, "tenant-1", "conv-new")
	require.NoError(t, err)
	require.Len(t, list, 1, "the note must be readable from the same conversation")
}

// Promotion resolves a contact from a conversation ID, so it must refuse a
// conversation belonging to another tenant — otherwise a note (and a new lead)
// would land on someone else's contact.
func TestNotesAdapter_AddNote_RefusesConversationOfAnotherTenant(t *testing.T) {
	f := newNotesAdapterTest()
	f.addConversation(t, "tenant-OTHER", "conv-x", "contact-9")

	_, err := f.adapter.AddNote(context.Background(), "tenant-1", "conv-x", "x", "user-1")
	require.Error(t, err)
	assert.Zero(t, f.creator.calls, "must not create a lead across tenants")
	assert.Empty(t, f.leadRepo.leads)
}

func TestNotesAdapter_AddNote_UnknownConversation(t *testing.T) {
	f := newNotesAdapterTest()
	_, err := f.adapter.AddNote(context.Background(), "tenant-1", "conv-inexistente", "x", "user-1")
	assert.ErrorIs(t, err, domain.ErrLeadNotFound)
}

// A tenant with no default funnel or entry column cannot have leads created.
// That failure reaches the operator on the note they just tried to save, instead
// of only the logs.
func TestNotesAdapter_AddNote_SurfacesLeadCreationFailure(t *testing.T) {
	f := newNotesAdapterTest()
	f.addConversation(t, "tenant-1", "conv-x", "contact-9")
	f.creator.err = errors.New("funil default nao configurado")

	_, err := f.adapter.AddNote(context.Background(), "tenant-1", "conv-x", "x", "user-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "funil default nao configurado")
}

func TestNotesAdapter_AddNote_EmptyContentRejected(t *testing.T) {
	ctx := context.Background()
	f := newNotesAdapterTest()

	lead, err := domain.NewLead(uuid.New().String(), "tenant-1", "f1", "c1", "contact-1", "conv-1")
	require.NoError(t, err)
	require.NoError(t, f.leadRepo.Create(ctx, lead))

	_, err = f.adapter.AddNote(ctx, "tenant-1", "conv-1", "", "user-1")
	assert.ErrorIs(t, err, domain.ErrNoteContentRequired)
}

func TestNotesAdapter_GenericLookupErrorPropagates(t *testing.T) {
	f := newNotesAdapterTest()
	f.leadRepo.forceErr = errors.New("db down")

	_, err := f.adapter.NotesForConversation(context.Background(), "tenant-1", "conv-1")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrLeadNotFound)
}
