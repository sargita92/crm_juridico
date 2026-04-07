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
	router     *gin.Engine
	provider   *authinfra.JWTProvider
	leadRepo   *owaspMockLeadRepo
	funnelRepo *owaspMockFunnelRepo
	columnRepo *owaspMockColumnRepo
	noteRepo   *owaspMockNoteRepo
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
func (m *owaspMockFunnelRepo) Update(_ context.Context, f *domain.Funnel) error { return nil }
func (m *owaspMockFunnelRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.Funnel, error) {
	return nil, nil
}
func (m *owaspMockFunnelRepo) FindDefaultByTenantID(_ context.Context, tenantID string) (*domain.Funnel, error) {
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
func (m *owaspMockColumnRepo) Update(_ context.Context, c *domain.Column) error   { return nil }
func (m *owaspMockColumnRepo) Delete(_ context.Context, id string) error           { return nil }
func (m *owaspMockColumnRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.Column, error) {
	return nil, nil
}
func (m *owaspMockColumnRepo) FindEntryByFunnelID(_ context.Context, funnelID string) (*domain.Column, error) {
	return nil, domain.ErrColumnNotFound
}
func (m *owaspMockColumnRepo) CountByFunnelID(_ context.Context, funnelID string) (int, error) {
	return 0, nil
}
func (m *owaspMockColumnRepo) GetMaxOrderIndex(_ context.Context, funnelID string) (int, error) {
	return 0, nil
}
func (m *owaspMockColumnRepo) SwapOrder(_ context.Context, col1ID string, order1 int, col2ID string, order2 int) error {
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
func (m *owaspMockLeadRepo) Update(_ context.Context, l *domain.Lead) error { return nil }
func (m *owaspMockLeadRepo) FindByContactAndTenant(_ context.Context, tenantID, contactID string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}
func (m *owaspMockLeadRepo) FindByFunnelID(_ context.Context, funnelID string, filter domain.LeadFilter) (*domain.LeadList, error) {
	return &domain.LeadList{}, nil
}
func (m *owaspMockLeadRepo) CountByColumnID(_ context.Context, columnID string) (int, error) {
	return 0, nil
}

func (m *owaspMockLeadRepo) FindByConversationID(_ context.Context, conversationID string) (*domain.Lead, error) {
	return nil, domain.ErrLeadNotFound
}

type owaspMockMovementRepo struct{}

func (m *owaspMockMovementRepo) Create(_ context.Context, mv *domain.LeadMovement) error { return nil }
func (m *owaspMockMovementRepo) FindByLeadID(_ context.Context, leadID string) ([]domain.LeadMovement, error) {
	return nil, nil
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
	return nil, nil
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
	movementRepo := &owaspMockMovementRepo{}
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
	createLeadUC := application.NewCreateLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo, nil, nil)
	moveLeadUC := application.NewMoveLeadUseCase(funnelRepo, columnRepo, leadRepo, movementRepo)
	userNameProvider := &owaspMockUserNameProvider{}
	getLeadDetailUC := application.NewGetLeadDetailUseCase(leadRepo, movementRepo, funnelRepo, columnRepo, contactProvider, messageProvider, noteRepo, userNameProvider, nil)
	createLeadNoteUC := application.NewCreateLeadNoteUseCase(leadRepo, noteRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(
		getKanbanUC, listFunnelsUC, getFunnelUC,
		createFunnelUC, updateFunnelUC, toggleFunnelUC,
		createColumnUC, deleteColumnUC, moveColumnUC,
		createLeadUC, moveLeadUC, getLeadDetailUC,
		createLeadNoteUC,
		leadRepo, nil, nil, testLog,
	)

	router := gin.New()

	tmpl := template.New("")
	for _, name := range []string{
		"funnel/kanban.html", "funnel/kanban_content.html",
		"funnel/lead_drawer.html", "funnel/lead_notes_section.html",
		"funnel/lead_move.html", "funnel/funnel_list.html",
		"funnel/funnel_detail.html", "funnel/funnel_form.html",
		"funnel/columns_section.html", "funnel/column_form.html",
		"funnel/lead_product_form.html", "funnel/lead_product_section.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &owaspEnv{
		router:     router,
		provider:   jwtProvider,
		leadRepo:   leadRepo,
		funnelRepo: funnelRepo,
		columnRepo: columnRepo,
		noteRepo:   noteRepo,
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
