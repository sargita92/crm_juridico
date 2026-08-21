package http

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
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

// The note field is unconditional. It used to hang off a HasLead flag, and a
// conversation without a lead lost the field entirely — indistinguishable on
// screen from "no notes yet", so notes read as broken. Every state below must
// keep the field.
func TestNotesPanel_AlwaysOffersTheNoteField(t *testing.T) {
	cases := []struct {
		name    string
		service *mockNotesService
	}{
		{"conversa sem nota nenhuma", &mockNotesService{}},
		{"conversa com notas", &mockNotesService{notes: []domain.ConversationNote{{Content: "oi"}}}},
		{"falha ao carregar", &mockNotesService{listErr: errors.New("boom")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deps := setupNotesRealTemplate(t)
			deps.handler.SetNotesService(tc.service)

			assert.Contains(t, renderNotesPanel(t, deps), "<textarea",
				"operator has no way to write a note")
		})
	}
}

// A failure loading the notes still has to say so — degrading to a silent empty
// panel would hide the outage behind a normal-looking screen.
func TestNotesPanel_LoadErrorIsVisible(t *testing.T) {
	deps := setupNotesRealTemplate(t)
	deps.handler.SetNotesService(&mockNotesService{listErr: errors.New("boom")})

	w := httptest.NewRecorder()
	deps.router.ServeHTTP(w, makeRequest(http.MethodGet, "/tenant/whatsapp/conversations/c1/notes"))

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Erro ao carregar notas")
}

func TestNotesPanel_EmptyStateWhenNoNotes(t *testing.T) {
	deps := setupNotesRealTemplate(t)
	deps.handler.SetNotesService(&mockNotesService{})

	assert.Contains(t, renderNotesPanel(t, deps), "Nenhuma anotacao")
}
