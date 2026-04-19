package http

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/files/application"
	"github.com/sasrgita/crm-juridico/internal/files/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
)

const testTenantID = "tenant-under-test"

// mockFileRepo (in-memory) used by handler tests.
type mockFileRepo struct {
	byID map[string]*domain.File
}

func newMockFileRepo() *mockFileRepo {
	return &mockFileRepo{byID: make(map[string]*domain.File)}
}
func (m *mockFileRepo) Create(_ context.Context, f *domain.File) error {
	cp := *f
	m.byID[f.ID] = &cp
	return nil
}
func (m *mockFileRepo) FindByID(_ context.Context, tenantID, id string) (*domain.File, error) {
	f, ok := m.byID[id]
	if !ok || f.TenantID != tenantID {
		return nil, domain.ErrFileNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *mockFileRepo) List(_ context.Context, q domain.ListQuery) (*domain.ListResult, error) {
	var items []domain.FileWithContext
	for _, f := range m.byID {
		if f.TenantID != q.TenantID {
			continue
		}
		if q.LeadID != nil && (f.LeadID == nil || *f.LeadID != *q.LeadID) {
			continue
		}
		if q.MediaType != nil && f.MediaType != *q.MediaType {
			continue
		}
		items = append(items, domain.FileWithContext{File: *f, ContactName: "Contato"})
	}
	return &domain.ListResult{
		Items:    items,
		Total:    int64(len(items)),
		Page:     1,
		PageSize: domain.DefaultPageSize,
	}, nil
}
func (m *mockFileRepo) CountByLead(_ context.Context, _ string, _ string) (int64, error) {
	return 0, nil
}
func (m *mockFileRepo) ListRecentByLead(_ context.Context, _ string, _ string, _ int) ([]domain.File, error) {
	return nil, nil
}

// mockStorage with a fixed payload.
type mockStorage struct {
	content []byte
}

func (m *mockStorage) Save(_ context.Context, _ string, content []byte) error {
	m.content = content
	return nil
}
func (m *mockStorage) Open(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.content)), nil
}
func (m *mockStorage) Size(_ context.Context, _ string) (int64, error) {
	return int64(len(m.content)), nil
}

// helpers to set up a router wired with the handler and real templates.
func setupHandlerEnv(t *testing.T) (*gin.Engine, *mockFileRepo, *mockStorage) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newMockFileRepo()
	storage := &mockStorage{content: []byte("payload")}

	listUC := application.NewListFilesUseCase(repo)
	getUC := application.NewGetFileUseCase(repo)
	downloadUC := application.NewDownloadFileUseCase(repo, storage)
	summaryUC := application.NewLeadFilesSummaryUseCase(repo)

	h := NewHandler(listUC, getUC, downloadUC, summaryUC, zap.NewNop())

	router := gin.New()

	tmpl := template.Must(template.New("").Funcs(testhelper.TemplateFuncMap()).ParseGlob(testhelper.TemplatesPath()))
	router.SetHTMLTemplate(tmpl)

	router.Use(func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	files := router.Group("/tenant/files")
	files.GET("", h.ListPage)
	files.GET("/list", h.ListFragment)
	files.GET("/:id/preview", h.PreviewDrawer)
	files.GET("/:id/download", h.Download)
	files.GET("/:id/thumbnail", h.Thumbnail)

	leads := router.Group("/tenant/leads")
	leads.GET("/:id/files-summary", h.LeadFilesSummary)

	return router, repo, storage
}

func mkTestFile(t *testing.T, tenantID, name string, mt domain.MediaType, mime string) *domain.File {
	t.Helper()
	f, err := domain.NewFile(
		"file-"+name, tenantID, "conv-1", "contact-1",
		name, mime, "k/"+name, 10, mt,
		domain.DirectionInbound, nil, nil,
	)
	require.NoError(t, err)
	return f
}

func doRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
	return w
}

func TestHandler_ListPage_EmptyStateRenders(t *testing.T) {
	router, _, _ := setupHandlerEnv(t)
	w := doRequest(router, http.MethodGet, "/tenant/files")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Ainda não há arquivos")
}

func TestHandler_ListPage_ListsFiles(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	require.NoError(t, repo.Create(context.Background(), mkTestFile(t, testTenantID, "contrato.pdf", domain.MediaTypeDocument, "application/pdf")))
	w := doRequest(router, http.MethodGet, "/tenant/files")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "contrato.pdf")
}

func TestHandler_ListFragment_FilterByType(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	require.NoError(t, repo.Create(context.Background(), mkTestFile(t, testTenantID, "img.jpg", domain.MediaTypeImage, "image/jpeg")))
	require.NoError(t, repo.Create(context.Background(), mkTestFile(t, testTenantID, "doc.pdf", domain.MediaTypeDocument, "application/pdf")))

	w := doRequest(router, http.MethodGet, "/tenant/files/list?type=image")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "img.jpg")
	assert.NotContains(t, w.Body.String(), "doc.pdf")
}

func TestHandler_ListFragment_InvalidType_400(t *testing.T) {
	router, _, _ := setupHandlerEnv(t)
	w := doRequest(router, http.MethodGet, "/tenant/files/list?type=bogus")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_PreviewDrawer_ReturnsMetadata(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	f := mkTestFile(t, testTenantID, "contrato.pdf", domain.MediaTypeDocument, "application/pdf")
	require.NoError(t, repo.Create(context.Background(), f))

	w := doRequest(router, http.MethodGet, "/tenant/files/"+f.ID+"/preview")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "contrato.pdf")
}

func TestHandler_PreviewDrawer_CrossTenant_404(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	f := mkTestFile(t, "other-tenant", "x.pdf", domain.MediaTypeDocument, "application/pdf")
	require.NoError(t, repo.Create(context.Background(), f))

	w := doRequest(router, http.MethodGet, "/tenant/files/"+f.ID+"/preview")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Download_SetsHeadersAndBody(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	f := mkTestFile(t, testTenantID, "contrato.pdf", domain.MediaTypeDocument, "application/pdf")
	require.NoError(t, repo.Create(context.Background(), f))

	w := doRequest(router, http.MethodGet, "/tenant/files/"+f.ID+"/download")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), "contrato.pdf")
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "payload", w.Body.String())
}

func TestHandler_Download_CrossTenant_404(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	f := mkTestFile(t, "other", "x.pdf", domain.MediaTypeDocument, "application/pdf")
	require.NoError(t, repo.Create(context.Background(), f))

	w := doRequest(router, http.MethodGet, "/tenant/files/"+f.ID+"/download")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Thumbnail_OnlyForImages(t *testing.T) {
	router, repo, _ := setupHandlerEnv(t)
	doc := mkTestFile(t, testTenantID, "doc.pdf", domain.MediaTypeDocument, "application/pdf")
	img := mkTestFile(t, testTenantID, "foto.jpg", domain.MediaTypeImage, "image/jpeg")
	require.NoError(t, repo.Create(context.Background(), doc))
	require.NoError(t, repo.Create(context.Background(), img))

	w := doRequest(router, http.MethodGet, "/tenant/files/"+doc.ID+"/thumbnail")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = doRequest(router, http.MethodGet, "/tenant/files/"+img.ID+"/thumbnail")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/jpeg", w.Header().Get("Content-Type"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestHandler_LeadFilesSummary_EmptyRenders(t *testing.T) {
	router, _, _ := setupHandlerEnv(t)
	w := doRequest(router, http.MethodGet, "/tenant/leads/some-lead/files-summary")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Nenhum arquivo associado")
}

func TestHandler_InvalidDateRange_400(t *testing.T) {
	router, _, _ := setupHandlerEnv(t)
	w := doRequest(router, http.MethodGet, "/tenant/files/list?period=custom&from=2026-04-10&to=2026-04-01")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
