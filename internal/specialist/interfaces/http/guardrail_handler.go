package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type GuardrailHandler struct {
	createUC *application.CreateGuardrailUseCase
	updateUC *application.UpdateGuardrailUseCase
	toggleUC *application.ToggleGuardrailUseCase
	deleteUC *application.DeleteGuardrailUseCase
	listUC   *application.ListGuardrailsUseCase
}

func NewGuardrailHandler(
	createUC *application.CreateGuardrailUseCase,
	updateUC *application.UpdateGuardrailUseCase,
	toggleUC *application.ToggleGuardrailUseCase,
	deleteUC *application.DeleteGuardrailUseCase,
	listUC *application.ListGuardrailsUseCase,
) *GuardrailHandler {
	return &GuardrailHandler{
		createUC: createUC, updateUC: updateUC,
		toggleUC: toggleUC, deleteUC: deleteUC, listUC: listUC,
	}
}

func (h *GuardrailHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/guardrails", h.HandleList)
	admin.POST("/:id/guardrails", h.HandleCreate)
	admin.PUT("/:id/guardrails/:gid", h.HandleUpdate)
	admin.POST("/:id/guardrails/:gid/toggle", h.HandleToggle)
	admin.DELETE("/:id/guardrails/:gid", h.HandleDelete)
}

func (h *GuardrailHandler) HandleList(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	items, err := h.listUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar guardrails"})
		return
	}
	c.HTML(http.StatusOK, "specialist/guardrails_section.html", gin.H{
		"Guardrails":   items,
		"SpecialistID": id,
	})
}

func (h *GuardrailHandler) HandleCreate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	input := application.CreateGuardrailInput{
		SpecialistID: id,
		Name:         c.PostForm("name"),
		Type:         c.PostForm("type"),
		Rule:         c.PostForm("rule"),
		Message:      c.PostForm("message"),
	}

	_, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, domain.ErrSpecialistInactive) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapGuardrailError(err)})
		return
	}

	h.HandleList(c)
}

func (h *GuardrailHandler) HandleUpdate(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	input := application.UpdateGuardrailInput{
		ID:      gid,
		Name:    c.PostForm("name"),
		Type:    c.PostForm("type"),
		Rule:    c.PostForm("rule"),
		Message: c.PostForm("message"),
	}

	_, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapGuardrailError(err)})
		return
	}

	h.HandleList(c)
}

func (h *GuardrailHandler) HandleToggle(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	if err := h.toggleUC.Execute(c.Request.Context(), gid); err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao alternator guardrail"})
		return
	}

	h.HandleList(c)
}

func (h *GuardrailHandler) HandleDelete(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	if err := h.deleteUC.Execute(c.Request.Context(), gid); err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir guardrail"})
		return
	}

	h.HandleList(c)
}

func mapGuardrailError(err error) string {
	switch {
	case errors.Is(err, domain.ErrGuardrailNameRequired):
		return "Nome é obrigatório"
	case errors.Is(err, domain.ErrGuardrailNameTooLong):
		return "Nome excede o tamanho máximo"
	case errors.Is(err, domain.ErrGuardrailRuleRequired):
		return "Regra é obrigatória"
	case errors.Is(err, domain.ErrGuardrailRuleTooLong):
		return "Regra excede o tamanho máximo"
	case errors.Is(err, domain.ErrGuardrailMessageTooLong):
		return "Mensagem excede o tamanho máximo"
	case errors.Is(err, domain.ErrGuardrailTypeInvalid):
		return "Tipo de guardrail inválido"
	default:
		return "Erro interno"
	}
}
