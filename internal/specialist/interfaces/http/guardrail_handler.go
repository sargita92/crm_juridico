package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// guardrailTypes é a lista canônica de opções exibidas no select Tipo.
// Mantida no backend para reutilizar entre create e edit e evitar divergência
// com isValidGuardrailType do domínio.
var guardrailTypes = []struct{ Value, Label string }{
	{"forbidden_topics", "Tópicos proibidos"},
	{"scope_limit", "Limite de escopo"},
	{"response_tone", "Tom de resposta"},
	{"security_lgpd", "Segurança / LGPD"},
	{"human_escalation", "Escalonamento humano"},
	{"output_validation", "Validação de saída"},
}

type GuardrailHandler struct {
	createUC        *application.CreateGuardrailUseCase
	updateUC        *application.UpdateGuardrailUseCase
	toggleUC        *application.ToggleGuardrailUseCase
	listUC          *application.ListGuardrailsUseCase
	attachUC        *application.AttachGuardrailUseCase
	detachUC        *application.DetachGuardrailUseCase
	listAvailableUC *application.ListAvailableGuardrailsUseCase
	guardrailRepo   domain.GuardrailRepository
}

func NewGuardrailHandler(
	createUC *application.CreateGuardrailUseCase,
	updateUC *application.UpdateGuardrailUseCase,
	toggleUC *application.ToggleGuardrailUseCase,
	listUC *application.ListGuardrailsUseCase,
	attachUC *application.AttachGuardrailUseCase,
	detachUC *application.DetachGuardrailUseCase,
	listAvailableUC *application.ListAvailableGuardrailsUseCase,
	guardrailRepo domain.GuardrailRepository,
) *GuardrailHandler {
	return &GuardrailHandler{
		createUC: createUC, updateUC: updateUC, toggleUC: toggleUC, listUC: listUC,
		attachUC: attachUC, detachUC: detachUC, listAvailableUC: listAvailableUC,
		guardrailRepo: guardrailRepo,
	}
}

func (h *GuardrailHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/guardrails", h.HandleList)
	admin.GET("/:id/guardrails/new/form", h.RenderCreateForm)
	admin.GET("/:id/guardrails/available", h.RenderAttachList)
	admin.GET("/:id/guardrails/:gid/form", h.RenderEditForm)
	admin.POST("/:id/guardrails", h.HandleCreate)
	admin.POST("/:id/guardrails/:gid/attach", h.HandleAttach)
	admin.PUT("/:id/guardrails/:gid", h.HandleUpdate)
	admin.POST("/:id/guardrails/:gid/toggle", h.HandleToggle)
	// DELETE here means DETACH from this specialist — the library item survives.
	admin.DELETE("/:id/guardrails/:gid", h.HandleDetach)
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

// RenderCreateForm responde GET /admin/specialists/:id/guardrails/new/form com
// o partial guardrail_form.html vazio, consumido pelo #guardrail-modal-body.
func (h *GuardrailHandler) RenderCreateForm(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	c.HTML(http.StatusOK, "specialist/guardrail_form.html", gin.H{
		"SpecialistID": id,
		"Types":        guardrailTypes,
	})
}

// RenderEditForm responde GET /admin/specialists/:id/guardrails/:gid/form com o
// partial guardrail_form.html pré-preenchido pelo guardrail persistido.
func (h *GuardrailHandler) RenderEditForm(c *gin.Context) {
	id := c.Param("id")
	gid := c.Param("gid")
	if !validUUID(id) || !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	g, err := h.guardrailRepo.FindByID(c.Request.Context(), gid)
	if err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar guardrail"})
		return
	}
	c.HTML(http.StatusOK, "specialist/guardrail_form.html", gin.H{
		"SpecialistID": id,
		"Types":        guardrailTypes,
		"Guardrail": gin.H{
			"ID":      g.ID,
			"Name":    g.Name,
			"Type":    string(g.Type),
			"Rule":    g.Rule,
			"Message": g.Message,
		},
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
			respondGuardrailFormError(c, http.StatusNotFound, "Especialista não encontrado.")
			return
		}
		if errors.Is(err, domain.ErrSpecialistInactive) {
			respondGuardrailFormError(c, http.StatusBadRequest, "Especialista está inativo. Reative antes de adicionar guardrails.")
			return
		}
		respondGuardrailFormError(c, http.StatusBadRequest, mapGuardrailError(err))
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
			respondGuardrailFormError(c, http.StatusNotFound, "Guardrail não encontrado.")
			return
		}
		respondGuardrailFormError(c, http.StatusBadRequest, mapGuardrailError(err))
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

// HandleDetach removes the guardrail from this specialist (deletes the join row).
// The library item itself is untouched and may still serve other specialists.
func (h *GuardrailHandler) HandleDetach(c *gin.Context) {
	id := c.Param("id")
	gid := c.Param("gid")
	if !validUUID(id) || !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	if err := h.detachUC.Execute(c.Request.Context(), id, gid); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao remover guardrail"})
		return
	}

	h.HandleList(c)
}

// RenderAttachList renders the picker of library guardrails not yet attached to
// this specialist, consumed by the "Anexar existente" modal.
func (h *GuardrailHandler) RenderAttachList(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	items, err := h.listAvailableUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar guardrails"})
		return
	}
	c.HTML(http.StatusOK, "specialist/guardrail_attach_list.html", gin.H{
		"Available":    items,
		"SpecialistID": id,
	})
}

// HandleAttach links an existing library guardrail to this specialist.
func (h *GuardrailHandler) HandleAttach(c *gin.Context) {
	id := c.Param("id")
	gid := c.Param("gid")
	if !validUUID(id) || !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	if err := h.attachUC.Execute(c.Request.Context(), id, gid); err != nil {
		switch {
		case errors.Is(err, domain.ErrSpecialistNotFound):
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
		case errors.Is(err, domain.ErrGuardrailNotFound):
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
		case errors.Is(err, domain.ErrSpecialistInactive):
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
		case errors.Is(err, domain.ErrGuardrailAlreadyAttached):
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "Guardrail já anexado"})
		default:
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao anexar guardrail"})
		}
		return
	}

	h.HandleList(c)
}

// respondGuardrailFormError redireciona o alerta HTML para o slot
// #guardrail-form-error dentro do modal — mesmo padrão dos steps: o
// hx-target do form aponta para #guardrails-section, então o erro precisa
// ser retargetado pra não corromper a lista.
func respondGuardrailFormError(c *gin.Context, status int, message string) {
	c.Header("HX-Retarget", "#guardrail-form-error")
	c.Header("HX-Reswap", "innerHTML")
	c.HTML(status, "specialist/guardrail_form_error.html", gin.H{"Message": message})
	c.Abort()
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
