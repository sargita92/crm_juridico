package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"

	landinghttp "github.com/sasrgita/crm-juridico/internal/landing/interfaces/http"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter() *gin.Engine {
	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	handler := landinghttp.NewHandler()
	handler.RegisterRoutes(router)

	return router
}

func TestGetLandingPage(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "CRM Jurídico")
	assert.Contains(t, w.Body.String(), "WhatsApp")
}

func TestPostContato(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/contato", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Obrigado")
}

func TestLandingPageContainsExpectedSections(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "hero")
	assert.Contains(t, body, "features")
	assert.Contains(t, body, "contato")
	assert.Contains(t, body, "/auth/login")
}
