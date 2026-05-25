package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

type owaspEnv struct {
	router       *gin.Engine
	provider     *authinfra.JWTProvider
	leadRepo     *owaspMockLeadRepo
	funnelRepo   *owaspMockFunnelRepo
	columnRepo   *owaspMockColumnRepo
	noteRepo     *owaspMockNoteRepo
	movementRepo *owaspMockMovementRepo
}

// Minimal mocks for OWASP tests (only what's needed for routing)

type owaspMockFunnelRepo struct{ funnels map[string]*domain.Funnel }

func (m *owaspMockFunnelRepo) Create(_ context.Context, f *domain.Funnel) error {
	m.funnels[f.ID] = f
	return nil
}
func (m *owaspMockFunnelRepo) FindByID(_ context.Context, id string) (*domain.Funnel, error) {
	if f, ok := m.funnels[id]; ok {
		return f, nil
	}
	return nil, domain.ErrFunnelNotFound
}
func (m *owaspMockFunnelRepo) Update(_ context.Context, f *domain.Funnel) error {
	m.funnels[f.ID] = f
	return nil
}
func (m *owaspMockFunnelRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.Funnel, error) {
	out := make([]domain.Funnel, 0)
	for _, f := range m.funnels {
		if f.TenantID == tenantID {
			out = append(out, *f)
		}
	}
	return out, nil
}
func (m *owaspMockFunnelRepo) FindDefaultByTenantID(_ context.Context, tenantID string) (*domain.Funnel, error) {
	for _, f := range m.funnels {
		if f.TenantID == tenantID && f.IsDefault {
			return f, nil
		}
	}
	return nil, domain.ErrFunnelNotFound
}

type owaspMockColumnRepo struct{ columns map[string]*domain.Column }

func (m *owaspMockColumnRepo) Create(_ context.Context, c *domain.Column) error {
	m.columns[c.ID] = c
	return nil
}
func (m *owaspMockColumnRepo) FindByID(_ context.Context, id string) (*domain.Column, error) {
	if c, ok := m.columns[id]; ok {
		return c, nil
	}
	return nil, domain.ErrColumnNotFound
}
func (m *owaspMockColumnRepo) Update(_ context.Context, c *domain.Column) error {
	m.columns[c.ID] = c
	return nil
}
func (m *owaspMockColumnRepo) Delete(_ context.Context, id string) error {
	delete(m.columns, id)
	return nil
}
func (m *owaspMockColumnRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.Column, error) {
	out := make([]domain.Column, 0)
	for _, c := range m.columns {
		if c.FunnelID == funnelID {
			out = append(out, *c)
		}
	}
	// Sort by order_index for stable iteration
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].OrderIndex < out[i].OrderIndex {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}
func (m *owaspMockColumnRepo) FindEntryByFunnelID(_ context.Context, funnelID string) (*domain.Column, error) {
	for _, c := range m.columns {
		if c.FunnelID == funnelID && c.Type == domain.ColumnTypeEntry {
			return c, nil
		}
	}
	return nil, domain.ErrColumnNotFound
}
func (m *owaspMockColumnRepo) CountByFunnelID(_ context.Context, funnelID string) (int, error) {
	count := 0
	for _, c := range m.columns {
		if c.FunnelID == funnelID {
			count++
		}
	}
	return count, nil
}
func (m *owaspMockColumnRepo) GetMaxOrderIndex(_ context.Context, funnelID string) (int, error) {
	max := -1
	for _, c := range m.columns {
		if c.FunnelID == funnelID && c.OrderIndex > max {
			max = c.OrderIndex
		}
	}
	if max < 0 {
		return 0, nil
	}
	return max, nil
}
func (m *owaspMockColumnRepo) SwapOrder(_ context.Context, col1ID string, order1 int, col2ID string, order2 int) error {
	if c1, ok := m.columns[col1ID]; ok {
		c1.OrderIndex = order1
	}
	if c2, ok := m.columns[col2ID]; ok {
		c2.OrderIndex = order2
	}
	return nil
}

type owaspMockLeadRepo struct{ leads map[string]*domain.Lead }

func (m *owaspMockLeadRepo) Create(_ context.Context, l *domain.Lead) error {
	m.leads[l.ID] = l
	return nil
}
func (m *owaspMockLeadRepo) FindByID(_ context.Context, id string) (*domain.Lead, error) {
	if l, ok := m.leads[id]; ok {
		return l, nil
	}
	return nil, domain.ErrLeadNotFound
}
func (m *owaspMockLeadRepo) Update(_ context.Context, l *domain.Lead) error {
	m.leads[l.ID] = l
	return nil
}
func (m *owaspMockLeadRepo) FindByContactAndTenant(_ context.Context, tenantID, contactID string) (*domain.Lead, error) {
	for _, l := range m.leads {
		if l.TenantID == tenantID && l.ContactID == contactID {
			return l, nil
		}
	}
	return nil, domain.ErrLeadNotFound
}
func (m *owaspMockLeadRepo) FindByFunnelID(_ context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) {
	list := &domain.LeadList{Page: filter.Page, Limit: filter.Limit}
	for _, l := range m.leads {
		if l.FunnelID != funnelID {
			continue
		}
		if filter.ColumnID != "" && l.ColumnID != filter.ColumnID {
			continue
		}
		list.Leads = append(list.Leads, domain.LeadWithContact{Lead: *l})
		list.Total++
	}
	return list, nil
}
func (m *owaspMockLeadRepo) CountByColumnID(_ context.Context, columnID string) (int, error) {
	count := 0
	for _, l := range m.leads {
		if l.ColumnID == columnID {
			count++
		}
	}
	return count, nil
}

func (m *owaspMockLeadRepo) FindByConversationID(_ context.Context, conversationID string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}

func (m *owaspMockLeadRepo) FindCurrentByConversationID(_ context.Context, _, conversationID string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}

func (m *owaspMockLeadRepo) FindByTenantAndSearch(_ context.Context, tenantID, query string, limit int) ([]domain.Lead, error) {
	return nil, nil
}

type owaspMockMovementRepo struct {
	movements map[string][]domain.LeadMovement
}

func (m *owaspMockMovementRepo) Create(_ context.Context, mv *domain.LeadMovement) error {
	if m.movements == nil {
		m.movements = make(map[string][]domain.LeadMovement)
	}
	m.movements[mv.LeadID] = append(m.movements[mv.LeadID], *mv)
	return nil
}
func (m *owaspMockMovementRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadMovement, error) {
	if m.movements == nil {
		return nil, nil
	}
	return m.movements[leadID], nil
}

type owaspMockContactProvider struct{}

func (m *owaspMockContactProvider) FindByID(_ context.Context, id string) (domain.ContactInfo, error) {
	return domain.ContactInfo{}, nil
}

type owaspMockMessageProvider struct{}

func (m *owaspMockMessageProvider) FindRecentByConversationID(_ context.Context, id string, limit int) ([]domain.MessageSummary, error) {
	return nil, nil
}

type owaspMockNoteRepo struct{ notes map[string][]*domain.LeadNote }

func (m *owaspMockNoteRepo) Create(_ context.Context, n *domain.LeadNote) error {
	m.notes[n.LeadID] = append(m.notes[n.LeadID], n)
	return nil
}
func (m *owaspMockNoteRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadNote, error) {
	out := make([]domain.LeadNote, 0)
	for _, n := range m.notes[leadID] {
		out = append(out, *n)
	}
	return out, nil
}

type owaspMockUserNameProvider struct{}

func (m *owaspMockUserNameProvider) FindNameByID(_ context.Context, id string) (string, error) {
	return "", nil
}

func setupOwaspEnv() *owaspEnv {
	gin.SetMode(gin.TestMode)

	funnelRepo := &owaspMockFunnelRepo{funnels: make(map[string]*domain.Funnel)}
	columnRepo := &owaspMockColumnRepo{columns: make(map[string]*domain.Column)}
	leadRepo := &owaspMockLeadRepo{leads: make(map[string]*domain.Lead)}
	movementRepo := &owaspMockMovementRepo{movements: make(map[string][]domain.LeadMovement)}
	contactProvider := &owaspMockContactProvider{}
	messageProvider := &owaspMockMessageProvider{}
	noteRepo := &owaspMockNoteRepo{notes: make(map[string][]*domain.LeadNote)}

	getKanbanUC := application.NewGetKanbanUseCase(funnelRepo, columnRepo, leadRepo)
	listFunnelsUC := application.NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)
	getFunnelUC := application.NewGetFunnelUseCase(funnelRepo, columnRepo)
	createFunnelUC := application.NewCreateFunnelUseCase(funnelRepo, columnRepo)
	updateFunnelUC := application.NewUpdateFunnelUseCase(funnelRepo)
	toggleFunnelUC := application.NewToggleFunnelUseCase(funnelRepo)
	createColumnUC := application.NewCreateColumnUseCase(funnelRepo, columnRepo)
	deleteColumnUC := application.NewDeleteColumnUseCase(funnelRepo, columnRepo, leadRepo)
	moveColumnUC := application.NewMoveColumnUseCase(funnelRepo, columnRepo)
	createLeadUC := application.NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil, nil, owaspStubPicker{})
	moveLeadUC := application.NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil)
	userNameProvider := &owaspMockUserNameProvider{}
	getLeadDetailUC := application.NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo, userNameProvider, nil)
	createLeadNoteUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)
	assignLeadUC := application.NewAssignLeadUseCase(leadRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC, assignLeadUC,
		leadRepo, nil, nil, testLog,
	)

	router := gin.New()

	tmpl := template.New("")
	for _, name := range []string{
		"funnel/kanban_content.html",
		"funnel/lead_drawer.html", "funnel/lead_notes_section.html",
		"funnel/lead_move.html", "funnel/funnel_list.html",
		"funnel/funnel_detail.html", "funnel/funnel_form.html",
		"funnel/columns_section.html", "funnel/column_form.html",
		"funnel/lead_product_form.html", "funnel/lead_product_section.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	// kanban.html stub renders OpenLeadID so deep-link tests can assert on it.
	template.Must(tmpl.New("funnel/kanban.html").Parse("ok{{if .OpenLeadID}} open:{{.OpenLeadID}}{{end}}"))
	router.SetHTMLTemplate(tmpl)

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &owaspEnv{
		router:       router,
		provider:     jwtProvider,
		leadRepo:     leadRepo,
		funnelRepo:   funnelRepo,
		columnRepo:   columnRepo,
		noteRepo:     noteRepo,
		movementRepo: movementRepo,
	}
}

func (e *owaspEnv) tenantToken(t *testing.T, tenantID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser, TenantID: tenantID,
	})
	require.NoError(t, err)
	return token
}

func (e *owaspEnv) tokenWithoutTenant(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(), Role: authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func owaspCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

func TestOWASP_A01_NoToken_Returns401(t *testing.T) {
	env := setupOwaspEnv()
	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/leads"},
		{http.MethodGet, "/tenant/leads/some-id"},
		{http.MethodPost, "/tenant/leads/some-id/notes"},
		{http.MethodPost, "/tenant/leads/some-id/move"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestOWASP_A01_NoTenant_Returns403(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tokenWithoutTenant(t)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/leads"},
		{http.MethodGet, "/tenant/leads/some-id"},
		{http.MethodPost, "/tenant/leads/some-id/notes"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			req.AddCookie(owaspCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestOWASP_A01_TenantIsolation_LeadNotVisible(t *testing.T) {
	env := setupOwaspEnv()

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-a", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = env.leadRepo.Create(context.Background(), lead)

	tokenB := env.tenantToken(t, "tenant-b")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/leads/"+lead.ID, nil)
	req.AddCookie(owaspCookie(tokenB))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// CT-64: Token com tenant adulterado (assinado por chave diferente) recebe 401
func TestOWASP_A07_TamperedTokenSignature_Returns401(t *testing.T) {
	env := setupOwaspEnv()

	// Generate a token signed with a DIFFERENT secret (simulando adulteracao)
	fakeProvider := authinfra.NewJWTProvider("another-secret", 24*time.Hour)
	tamperedToken, err := fakeProvider.Generate(authdomain.TokenClaims{
		UserID:   uuid.New().String(),
		Role:     authdomain.UserRoleUser,
		TenantID: "any-tenant",
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/leads", nil)
	req.AddCookie(owaspCookie(tamperedToken))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"token assinado com chave diferente deve ser rejeitado com 401")
}

// CT-64 (complemento): Token com payload modificado manualmente (assinatura invalida)
func TestOWASP_A07_InvalidTokenString_Returns401(t *testing.T) {
	env := setupOwaspEnv()

	// Token base valido
	validToken := env.tenantToken(t, "tenant-a")
	// Corrompe ultimo caractere (signature segment) com um valor garantido
	// diferente do original (senao a tampering vira no-op em ~1.5% dos runs).
	last := validToken[len(validToken)-1]
	replacement := byte('X')
	if last == replacement {
		replacement = 'Y'
	}
	tampered := validToken[:len(validToken)-1] + string(replacement)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/leads", nil)
	req.AddCookie(owaspCookie(tampered))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"token com assinatura corrompida deve ser rejeitado com 401")
}

func TestOWASP_A01_TenantIsolation_CreateNoteDenied(t *testing.T) {
	env := setupOwaspEnv()

	lead, _ := domain.NewLead(uuid.New().String(), "tenant-a", "funnel-1", "col-1", "contact-1", "conv-1")
	_ = env.leadRepo.Create(context.Background(), lead)

	tokenB := env.tenantToken(t, "tenant-b")
	body := strings.NewReader("content=test+note")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/leads/"+lead.ID+"/notes", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(owaspCookie(tokenB))
	env.router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

// owaspStubPicker keeps OWASP wiring working after CreateLeadUseCase gained a
// required ResponsiblePicker. These tests do not exercise lead creation, so a
// trivial stub is sufficient.
type owaspStubPicker struct{}

func (owaspStubPicker) PickForFunnel(_ context.Context, _, _, _ string) (domain.PickResult, error) {
	return domain.PickResult{UserID: "owasp-stub-user", Outcome: domain.PickOutcomePicked}, nil
}
