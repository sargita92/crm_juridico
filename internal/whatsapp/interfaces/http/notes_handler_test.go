package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type mockNotesService struct {
	hasLead bool
	notes   []domain.ConversationNote
	addErr  error
	listErr error
}

func (m *mockNotesService) NotesForConversation(_ context.Context, _, _ string) (bool, []domain.ConversationNote, error) {
	if m.listErr != nil {
		return false, nil, m.listErr
	}
	return m.hasLead, m.notes, nil
}

func (m *mockNotesService) AddNote(_ context.Context, _, _, content, _ string) ([]domain.ConversationNote, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	m.notes = append(m.notes, domain.ConversationNote{Content: content, AuthorName: "User", CreatedAt: time.Now()})
	return m.notes, nil
}

func TestRenderNotesPanel_HasLead(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{
		hasLead: true,
		notes:   []domain.ConversationNote{{Content: "oi"}},
	})

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderNotesPanel_NoLead(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{hasLead: false})

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderNotesPanel_ServiceError(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{listErr: errors.New("boom")})

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleCreateNote_Success(t *testing.T) {
	deps := setupTest()
	svc := &mockNotesService{hasLead: true}
	deps.handler.SetNotesService(svc)

	form := url.Values{"content": {"nota nova"}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", form.Encode()))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, svc.notes, 1)
}

func TestHandleCreateNote_Empty(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{hasLead: true})

	form := url.Values{"content": {""}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", form.Encode()))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleCreateNote_ServiceError(t *testing.T) {
	deps := setupTest()
	deps.handler.SetNotesService(&mockNotesService{hasLead: true, addErr: errors.New("boom")})

	form := url.Values{"content": {"x"}}
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", form.Encode()))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
