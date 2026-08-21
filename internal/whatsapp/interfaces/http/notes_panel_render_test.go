package http

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupNotesRealTemplate wires the notes routes against the REAL notes_panel.html
// instead of the "ok" stub used by the other handler tests, so assertions can look
// at what the operator actually sees in the drawer.
func setupNotesRealTemplate(t *testing.T) *testDeps {
	t.Helper()

	deps := setupTest()
	tmpl := template.Must(template.ParseFiles("../../../../web/templates/whatsapp/notes_panel.html"))

	router := gin.New()
	router.SetHTMLTemplate(tmpl)
	authMw := func(c *gin.Context) { c.Set("user_id", "user-1"); c.Next() }
	tenantMw := func(c *gin.Context) { c.Next() }
	deps.handler.RegisterRoutes(router, authMw, tenantMw)
	deps.router = router

	return deps
}

func renderNotesPanel(t *testing.T, deps *testDeps) string {
	t.Helper()
	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))
	return w.Body.String()
}

// A lead with no notes yet is the normal starting point of every conversation.
// The form has to be there — otherwise the first note can never be written.
func TestNotesPanel_LeadWithoutNotes_StillShowsForm(t *testing.T) {
	deps := setupNotesRealTemplate(t)
	deps.handler.SetNotesService(&mockNotesService{hasLead: true, notes: nil})

	body := renderNotesPanel(t, deps)

	assert.Contains(t, body, "<textarea", "no way to write the first note")
	assert.Contains(t, body, "Nenhuma anotacao")
}

// A failure loading the notes says nothing about whether the conversation has a
// lead. Rendering the "no lead" empty state there states something false and
// removes the form, so a transient database blip looks exactly like a broken
// feature — and the operator has no way to retry.
func TestNotesPanel_LoadError_KeepsFormAndDoesNotClaimNoLead(t *testing.T) {
	deps := setupNotesRealTemplate(t)
	deps.handler.SetNotesService(&mockNotesService{listErr: errors.New("boom")})

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))
	body := w.Body.String()

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, body, "Erro ao carregar notas")
	assert.NotContains(t, body, "wa-notes-blocked",
		"a load failure is not evidence that the conversation has no lead")
	assert.Contains(t, body, "<textarea", "operator lost the form to a transient error")
}

// The genuine no-lead case has nowhere to store a note, so the form is gone on
// purpose. It must then say so loudly: rendered as plain grey text like
// "Nenhuma anotacao" it is indistinguishable from the ordinary empty state, and
// the missing field reads as a broken feature instead of a missing lead.
func TestNotesPanel_NoLead_ExplainsWhyFormIsMissing(t *testing.T) {
	deps := setupNotesRealTemplate(t)
	deps.handler.SetNotesService(&mockNotesService{hasLead: false})

	body := renderNotesPanel(t, deps)

	assert.False(t, strings.Contains(body, "<textarea"),
		"a note has nowhere to be stored without a lead")
	assert.Contains(t, body, "wa-notes-blocked",
		"must not reuse the plain empty-state styling of 'Nenhuma anotacao'")
	assert.Contains(t, body, "ainda nao virou um", "must say why there is no field")
}
