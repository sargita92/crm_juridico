package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

// ---------- Helpers ----------

// seedFunnelWithColumns creates a funnel plus the default 5 columns, all
// belonging to the given tenant. Returns the funnel and the ordered columns.
// If isDefault is true the funnel is flagged as the tenant's default funnel.
func seedFunnelWithColumns(t *testing.T, env *owaspEnv, tenantID, name string, isDefault bool) (*domain.Funnel, []*domain.Column) {
	t.Helper()
	f, err := domain.NewFunnel(uuid.New().String(), tenantID, name, "desc")
	require.NoError(t, err)
	if isDefault {
		f.SetDefault()
	}
	require.NoError(t, env.funnelRepo.Create(context.Background(), f))

	cols := make([]*domain.Column, 0, len(domain.DefaultColumns))
	for i, dc := range domain.DefaultColumns {
		col, err := domain.NewColumn(uuid.New().String(), f.ID, dc.Name, i, dc.Type, dc.Color)
		require.NoError(t, err)
		require.NoError(t, env.columnRepo.Create(context.Background(), col))
		cols = append(cols, col)
	}
	return f, cols
}

// seedLead creates and stores a lead pointing at the given funnel/column.
func seedLead(t *testing.T, env *owaspEnv, tenantID, funnelID, columnID string) *domain.Lead {
	t.Helper()
	lead, err := domain.NewLead(
		uuid.New().String(), tenantID, funnelID, columnID,
		uuid.New().String(), uuid.New().String(),
	)
	require.NoError(t, err)
	require.NoError(t, env.leadRepo.Create(context.Background(), lead))
	return lead
}

// doReq issues a request against env.router with an auth cookie attached.
func doReq(env *owaspEnv, method, path, token string, body *strings.Reader, contentType string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, body)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.AddCookie(owaspCookie(token))
	env.router.ServeHTTP(w, req)
	return w
}

// ---------- Kanban ----------

func TestRenderKanbanPage_NoFunnels_Returns200(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestRenderKanbanPage_WithDefaultFunnel_Returns200(t *testing.T) {
	env := setupOwaspEnv()
	seedFunnelWithColumns(t, env, "tenant-1", "Default Funnel", true)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderKanbanPage_NonDefaultFunnel_FallsBackToFirst(t *testing.T) {
	env := setupOwaspEnv()
	// A non-default funnel is still picked as "current" when there is no default
	seedFunnelWithColumns(t, env, "tenant-1", "Funnel A", false)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderKanbanContent_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads/kanban?funnel_id="+f.ID, token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestRenderKanbanContent_UnknownFunnel_Returns200Empty(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	// Handler renders empty kanban on error (not 4xx) by design.
	w := doReq(env, http.MethodGet, "/tenant/leads/kanban?funnel_id=does-not-exist", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderKanbanContent_WithSearch(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "F", true)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet,
		"/tenant/leads/kanban?funnel_id="+f.ID+"&search=joao&product_id=prod-x",
		token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- Funnel list / form / detail ----------

func TestRenderFunnelList_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	seedFunnelWithColumns(t, env, "tenant-1", "First", true)
	seedFunnelWithColumns(t, env, "tenant-1", "Second", false)
	// Different tenant: must NOT leak into this tenant's listing
	seedFunnelWithColumns(t, env, "other-tenant", "Other", true)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads/funnels", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFunnelList_Empty(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFunnelForm_New(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/new", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFunnelForm_Edit_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "Edit Me", false)
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID+"/edit", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFunnelForm_Edit_NotFound(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/missing-id/edit", token, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenderFunnelDetail_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "Detail", false)
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID, token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFunnelDetail_WrongTenant_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-a", "TA Funnel", false)

	tokenB := env.tenantToken(t, "tenant-b")
	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID, tokenB, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenderFunnelDetail_Missing_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/does-not-exist", token, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- Create funnel ----------

func TestHandleCreateFunnel_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	body := strings.NewReader("name=New+Funnel&description=a+desc")
	w := doReq(env, http.MethodPost, "/tenant/leads/funnels", token, body,
		"application/x-www-form-urlencoded")

	// Handler uses c.Redirect(http.StatusFound, ...) on success.
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/tenant/leads/funnels", w.Header().Get("Location"))

	// Verify the funnel was persisted for this tenant
	funnels, err := env.funnelRepo.FindByTenantID(context.Background(), "tenant-1")
	require.NoError(t, err)
	assert.Len(t, funnels, 1)
	assert.Equal(t, "New Funnel", funnels[0].Name)
}

func TestHandleCreateFunnel_InvalidName_Returns422(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	body := strings.NewReader("name=&description=no+name")
	w := doReq(env, http.MethodPost, "/tenant/leads/funnels", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------- Update funnel ----------

func TestHandleUpdateFunnel_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "Old Name", false)
	token := env.tenantToken(t, "tenant-1")

	body := strings.NewReader("name=Renamed&description=updated")
	w := doReq(env, http.MethodPut, "/tenant/leads/funnels/"+f.ID, token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/tenant/leads/funnels/"+f.ID, w.Header().Get("HX-Redirect"))

	updated, err := env.funnelRepo.FindByID(context.Background(), f.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
}

func TestHandleUpdateFunnel_InvalidName_Returns422(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "Old Name", false)
	token := env.tenantToken(t, "tenant-1")

	body := strings.NewReader("name=&description=x")
	w := doReq(env, http.MethodPut, "/tenant/leads/funnels/"+f.ID, token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------- Toggle funnel ----------

func TestHandleToggleFunnel_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "Toggle", false)
	token := env.tenantToken(t, "tenant-1")

	require.True(t, f.Active)

	w := doReq(env, http.MethodPost, "/tenant/leads/funnels/"+f.ID+"/toggle", token, nil, "")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/tenant/leads/funnels/"+f.ID, w.Header().Get("Location"))

	updated, err := env.funnelRepo.FindByID(context.Background(), f.ID)
	require.NoError(t, err)
	assert.False(t, updated.Active, "funnel should be deactivated after toggle")

	// Toggling again flips it back.
	w2 := doReq(env, http.MethodPost, "/tenant/leads/funnels/"+f.ID+"/toggle", token, nil, "")
	assert.Equal(t, http.StatusFound, w2.Code)
	updated2, err := env.funnelRepo.FindByID(context.Background(), f.ID)
	require.NoError(t, err)
	assert.True(t, updated2.Active)
}

// ---------- Columns ----------

func TestRenderColumnsSection_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID+"/columns", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderColumnsSection_WrongTenant_Returns500(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-a", "F", false)
	tokenB := env.tenantToken(t, "tenant-b")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID+"/columns", tokenB, nil, "")

	// GetFunnel returns ErrFunnelNotFound on tenant mismatch; handler maps that to 500.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRenderColumnForm_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/funnels/"+f.ID+"/columns/new", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleCreateColumn_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	before, _ := env.columnRepo.CountByFunnelID(context.Background(), f.ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("name=Negociacao&type=intermediate&color=%23ff0000")
	w := doReq(env, http.MethodPost, "/tenant/leads/funnels/"+f.ID+"/columns", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)

	after, _ := env.columnRepo.CountByFunnelID(context.Background(), f.ID)
	assert.Equal(t, before+1, after, "column should have been appended")
}

func TestHandleCreateColumn_InvalidType_StillReRendersSection(t *testing.T) {
	env := setupOwaspEnv()
	f, _ := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	token := env.tenantToken(t, "tenant-1")

	// Invalid type string — use case returns error, handler still renders the
	// columns section with HTTP 200 (errors are logged, not surfaced).
	body := strings.NewReader("name=Bad&type=invalid-type&color=%23ff0000")
	w := doReq(env, http.MethodPost, "/tenant/leads/funnels/"+f.ID+"/columns", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleDeleteColumn_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	// Use a column with no leads so deletion is allowed.
	target := cols[1]
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodDelete,
		"/tenant/leads/funnels/"+f.ID+"/columns/"+target.ID, token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)

	_, err := env.columnRepo.FindByID(context.Background(), target.ID)
	assert.ErrorIs(t, err, domain.ErrColumnNotFound)
}

func TestHandleDeleteColumn_WithLeads_KeepsColumn(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	target := cols[0]
	// seed a lead in target column -> deletion should be blocked by the UC,
	// but the handler still renders the columns section (errors are logged only).
	seedLead(t, env, "tenant-1", f.ID, target.ID)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodDelete,
		"/tenant/leads/funnels/"+f.ID+"/columns/"+target.ID, token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
	// Column must still be there.
	_, err := env.columnRepo.FindByID(context.Background(), target.ID)
	assert.NoError(t, err)
}

func TestHandleMoveColumnUp_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	second := cols[1] // order_index 1, moving up -> should swap with cols[0]
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodPost,
		"/tenant/leads/funnels/"+f.ID+"/columns/"+second.ID+"/move-up", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)

	// second swapped to order_index 0
	updated, err := env.columnRepo.FindByID(context.Background(), second.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updated.OrderIndex)
}

func TestHandleMoveColumnDown_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	first := cols[0] // order_index 0, moving down -> swaps with cols[1]
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodPost,
		"/tenant/leads/funnels/"+f.ID+"/columns/"+first.ID+"/move-down", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := env.columnRepo.FindByID(context.Background(), first.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updated.OrderIndex)
}

func TestHandleMoveColumnUp_AlreadyFirst_RendersAnyway(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", false)
	first := cols[0]
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodPost,
		"/tenant/leads/funnels/"+f.ID+"/columns/"+first.ID+"/move-up", token, nil, "")

	// Handler logs the error but still renders the section with 200.
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------- Lead detail ----------

func TestRenderLeadDetail_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	// A movement entry gives the use case a non-empty Movements slice.
	_ = env.movementRepo.Create(context.Background(),
		domain.NewLeadMovement(uuid.New().String(), lead.ID, "", cols[0].ID))

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads/"+lead.ID, token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderLeadDetail_MissingLead_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/not-a-lead", token, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- Lead move form ----------

func TestRenderLeadMoveForm_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads/"+lead.ID+"/move-form", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderLeadMoveForm_MissingLead_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/does-not-exist/move-form", token, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------- Move lead ----------

func TestHandleMoveLead_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("column_id=" + cols[2].ID + "&funnel_id=" + f.ID)
	w := doReq(env, http.MethodPost, "/tenant/leads/"+lead.ID+"/move", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/tenant/leads", w.Header().Get("HX-Redirect"))

	updated, err := env.leadRepo.FindByID(context.Background(), lead.ID)
	require.NoError(t, err)
	assert.Equal(t, cols[2].ID, updated.ColumnID)
}

func TestHandleMoveLead_BadColumn_Returns422(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("column_id=not-a-column&funnel_id=" + f.ID)
	w := doReq(env, http.MethodPost, "/tenant/leads/"+lead.ID+"/move", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------- Create note ----------

func TestHandleCreateNote_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("content=this+is+a+note")
	w := doReq(env, http.MethodPost, "/tenant/leads/"+lead.ID+"/notes", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)

	notes, _ := env.noteRepo.FindByLeadID(context.Background(), lead.ID)
	assert.Len(t, notes, 1)
	assert.Equal(t, "this is a note", notes[0].Content)
}

func TestHandleCreateNote_EmptyContent_Returns422(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("content=")
	w := doReq(env, http.MethodPost, "/tenant/leads/"+lead.ID+"/notes", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------- Assign lead ----------

func TestHandleAssignLead_JSONBody_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader(`{"user_id":"user-42"}`)
	w := doReq(env, http.MethodPut, "/tenant/leads/"+lead.ID+"/assign", token, body,
		"application/json")

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := env.leadRepo.FindByID(context.Background(), lead.ID)
	require.NoError(t, err)
	assert.Equal(t, "user-42", updated.ResponsibleUserID)
}

func TestHandleAssignLead_FormBody_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("user_id=user-99")
	w := doReq(env, http.MethodPut, "/tenant/leads/"+lead.ID+"/assign", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := env.leadRepo.FindByID(context.Background(), lead.ID)
	require.NoError(t, err)
	assert.Equal(t, "user-99", updated.ResponsibleUserID)
}

func TestHandleAssignLead_WrongTenant_Returns422(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-a", "F", true)
	lead := seedLead(t, env, "tenant-a", f.ID, cols[0].ID)

	tokenB := env.tenantToken(t, "tenant-b")
	body := strings.NewReader(`{"user_id":"u"}`)
	w := doReq(env, http.MethodPut, "/tenant/leads/"+lead.ID+"/assign", tokenB, body,
		"application/json")

	// AssignLeadUC returns ErrLeadNotFound, handler maps any UC error to 422.
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

// ---------- Lead product ----------

func TestRenderLeadProductForm_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	w := doReq(env, http.MethodGet, "/tenant/leads/"+lead.ID+"/product-form", token, nil, "")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderLeadProductForm_MissingLead_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := doReq(env, http.MethodGet, "/tenant/leads/does-not-exist/product-form", token, nil, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSetLeadProduct_HappyPath(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-1", "F", true)
	lead := seedLead(t, env, "tenant-1", f.ID, cols[0].ID)

	token := env.tenantToken(t, "tenant-1")
	body := strings.NewReader("product_id=prod-1")
	w := doReq(env, http.MethodPut, "/tenant/leads/"+lead.ID+"/product", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusOK, w.Code)

	updated, err := env.leadRepo.FindByID(context.Background(), lead.ID)
	require.NoError(t, err)
	assert.Equal(t, "prod-1", updated.ProductID)
}

func TestHandleSetLeadProduct_MissingLead_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	body := strings.NewReader("product_id=prod-1")
	w := doReq(env, http.MethodPut, "/tenant/leads/unknown/product", token, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleSetLeadProduct_WrongTenant_Returns404(t *testing.T) {
	env := setupOwaspEnv()
	f, cols := seedFunnelWithColumns(t, env, "tenant-a", "F", true)
	lead := seedLead(t, env, "tenant-a", f.ID, cols[0].ID)

	tokenB := env.tenantToken(t, "tenant-b")
	body := strings.NewReader("product_id=prod-1")
	w := doReq(env, http.MethodPut, "/tenant/leads/"+lead.ID+"/product", tokenB, body,
		"application/x-www-form-urlencoded")

	assert.Equal(t, http.StatusNotFound, w.Code)
}
