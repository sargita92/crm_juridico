package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/", h.index)
	router.POST("/contato", h.contato)
}

func (h *Handler) index(c *gin.Context) {
	c.HTML(http.StatusOK, "landing/index.html", gin.H{
		"Year": time.Now().Year(),
	})
}

func (h *Handler) contato(c *gin.Context) {
	c.HTML(http.StatusOK, "landing/contact_success.html", nil)
}
