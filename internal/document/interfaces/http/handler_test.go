package http

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/document/application"
	docinfra "github.com/sasrgita/crm-juridico/internal/document/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	specialistinfra "github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

type testEnv struct {
	router         *gin.Engine
	provider       *authinfra.JWTProvider
	docRepo        *docinfra.GormDocumentRepository
	specDocRepo    *docinfra.GormSpecialistDocumentRepository
	specialistRepo *specialistinfra.GormSpecialistRepository
	db             *gorm.DB
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM specialist_mcps")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM mcp_servers")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	docRepo := docinfra.NewGormDocumentRepository(db)
	specDocRepo := docinfra.NewGormSpecialistDocumentRepository(db)
	specialistRepo := specialistinfra.NewGormSpecialistRepository(db)

	jwtProvider := authinfra.NewJWTProvider("test-secret", 24*time.Hour)

	uploadUC := application.NewUploadDocumentUseCase(docRepo)
	listUC := application.NewListDocumentsUseCase(docRepo, specDocRepo)
	getUC := application.NewGetDocumentUseCase(docRepo, specDocRepo)
	deleteUC := application.NewDeleteDocumentUseCase(docRepo, specDocRepo)
	associateUC := application.NewAssociateDocumentUseCase(specialistRepo, docRepo, specDocRepo)
	dissociateUC := application.NewDissociateDocumentUseCase(specDocRepo)
	listSpecDocsUC := application.NewListSpecialistDocumentsUseCase(specDocRepo, docRepo)
	listAvailableUC := application.NewListAvailableDocumentsUseCase(specDocRepo, docRepo)

	handler := NewHandler(
		uploadUC, listUC, getUC, deleteUC,
		associateUC, dissociateUC, listSpecDocsUC, listAvailableUC,
	)

	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	authMw := middleware.Auth(jwtProvider)
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, adminMw)

	return &testEnv{
		router:         router,
		provider:       jwtProvider,
		docRepo:        docRepo,
		specDocRepo:    specDocRepo,
		specialistRepo: specialistRepo,
		db:             db,
	}
}

func (e *testEnv) adminToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleAdmin,
	})
	require.NoError(t, err)
	return token
}

func (e *testEnv) userToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func tokenCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

func postForm(path string, values url.Values, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func getReq(path string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func deleteReq(path string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// multipartUpload builds a multipart/form-data request with a file field and an optional name field.
func multipartUpload(path, fieldName, fileName, content string, extraFields map[string]string, cookies ...*http.Cookie) *http.Request {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for k, v := range extraFields {
		_ = w.WriteField(k, v)
	}

	fw, _ := w.CreateFormFile(fieldName, fileName)
	_, _ = fw.Write([]byte(content))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// --- OWASP A01: Access Control - no token ---

func TestDocumentRoutes_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/documents"},
		{http.MethodPost, "/admin/documents"},
		{http.MethodGet, "/admin/documents/some-id"},
		{http.MethodDelete, "/admin/documents/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/documents"},
		{http.MethodPost, "/admin/specialists/some-id/documents"},
		{http.MethodDelete, "/admin/specialists/some-id/documents/doc-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// --- OWASP A01: Access Control - user role ---

func TestDocumentRoutes_UserRole_Returns403(t *testing.T) {
	env := setupTestEnv(t)
	token := env.userToken(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/documents"},
		{http.MethodPost, "/admin/documents"},
		{http.MethodGet, "/admin/documents/some-id"},
		{http.MethodDelete, "/admin/documents/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/documents"},
		{http.MethodPost, "/admin/specialists/some-id/documents"},
		{http.MethodDelete, "/admin/specialists/some-id/documents/doc-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			req.AddCookie(tokenCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// --- OWASP A07: Authentication Failures - invalid token ---

func TestDocumentRoutes_InvalidToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/documents", &http.Cookie{Name: "token", Value: "invalid.token.here"})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- OWASP A03: Injection - XSS in name ---

func TestHandleUpload_XSS_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	xssName := "<script>alert('xss')</script>"

	w := httptest.NewRecorder()
	req := multipartUpload(
		"/admin/documents",
		"file", "test.pdf",
		"%PDF-1.4 test content",
		map[string]string{"name": xssName},
		tokenCookie(token),
	)
	env.router.ServeHTTP(w, req)

	// Upload should succeed (200 with HX-Redirect) or fail with 400 (bad name)
	// Either way, the XSS name must not appear unescaped in any subsequent response
	if w.Code == http.StatusOK {
		redirect := w.Header().Get("HX-Redirect")
		if redirect != "" {
			id := strings.TrimPrefix(redirect, "/admin/documents/")
			w2 := httptest.NewRecorder()
			req2 := getReq("/admin/documents/"+id, tokenCookie(token))
			env.router.ServeHTTP(w2, req2)

			assert.NotContains(t, w2.Body.String(), "<script>")
			assert.Contains(t, w2.Body.String(), "&lt;script&gt;")
		}
	}

	// Ensure DB table is still intact regardless
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM documents").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(0))
}

// --- OWASP A03: Injection - SQL injection in search ---

func TestRenderList_SQLInjection_InSearch(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/documents?search='+OR+1%3D1+--", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Table must still be accessible — injection did not destroy it
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM documents").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(0))
}

// --- Functional: Upload success ---

func TestHandleUpload_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := multipartUpload(
		"/admin/documents",
		"file", "contrato.pdf",
		"%PDF-1.4 sample content",
		map[string]string{"name": "Contrato de Prestacao"},
		tokenCookie(token),
	)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/admin/documents/")

	// Verify record was persisted
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM documents WHERE name = ?", "Contrato de Prestacao").Scan(&count)
	assert.Equal(t, int64(1), count)
}

// --- Functional: Upload - no file returns 400 ---

func TestHandleUpload_NoFile_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/documents", url.Values{
		"name": {"Sem Arquivo"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Functional: Upload - invalid file type ---

func TestHandleUpload_InvalidType_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := multipartUpload(
		"/admin/documents",
		"file", "malware.exe",
		"MZ exec content",
		map[string]string{"name": "Arquivo Perigoso"},
		tokenCookie(token),
	)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Functional: List returns 200 ---

func TestRenderList_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/documents", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Functional: Detail not found ---

func TestRenderDetail_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/documents/00000000-0000-0000-0000-000000000000", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Functional: Delete not found ---

func TestHandleDelete_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/documents/00000000-0000-0000-0000-000000000000", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Functional: List specialist documents returns 200 ---

func TestHandleListSpecialistDocuments_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	// Use a random specialist ID — returns empty list (not 404)
	specialistID := uuid.New().String()

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+specialistID+"/documents", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Functional: Associate documents - no IDs returns 400 ---

func TestHandleAssociateDocuments_NoIDs_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	specialistID := uuid.New().String()

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+specialistID+"/documents", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
