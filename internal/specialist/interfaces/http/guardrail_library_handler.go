package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// GuardrailLibraryHandler serves the central guardrail library at /admin/guardrails,
// where reusable guardrails are created, edited, toggled and deleted independently
// of any specialist.
type GuardrailLibraryHandler struct {
	listAllUC *application.ListAllGuardrailsUseCase
	createUC  *application.CreateGuardrailUseCase
	updateUC  *application.UpdateGuardrailUseCase
	toggleUC  *application.ToggleGuardrailUseCase
	deleteUC  *application.DeleteGuardrailUseCase
	repo      domain.GuardrailRepository
}

func NewGuardrailLibraryHandler(
	listAllUC *application.ListAllGuardrailsUseCase,
	createUC *application.CreateGuardrailUseCase,
	updateUC *application.UpdateGuardrailUseCase,
	toggleUC *application.ToggleGuardrailUseCase,
	deleteUC *application.DeleteGuardrailUseCase,
	repo domain.GuardrailRepository,
) *GuardrailLibraryHandler {
	return &GuardrailLibraryHandler{
		listAllUC: listAllUC, createUC: createUC, updateUC: updateUC,
		toggleUC: toggleUC, deleteUC: deleteUC, repo: repo,
	}
}

func (h *GuardrailLibraryHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/guardrails")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderPage)
	admin.GET("/new/form", h.RenderCreateForm)
	admin.GET("/:gid/form", h.RenderEditForm)
	admin.POST("", h.HandleCreate)
	admin.PUT("/:gid", h.HandleUpdate)
	admin.POST("/:gid/toggle", h.HandleToggle)
	admin.DELETE("/:gid", h.HandleDelete)
}

func (h *GuardrailLibraryHandler) RenderPage(c *gin.Context) {
	items, err := h.listAllUC.Execute(c.Request.Context())
	if err != nil {
		c.HTML(http.StatusInternalServerError, "guardrail/library.html", gin.H{"Error": "Erro ao carregar guardrails"})
		return
	}
	c.HTML(http.StatusOK, "guardrail/library.html", gin.H{"Guardrails": items})
}

func (h *GuardrailLibraryHandler) renderSection(c *gin.Context, status int, errMsg string) {
	items, err := h.listAllUC.Execute(c.Request.Context())
	if err != nil {
		c.HTML(http.StatusInternalServerError, "guardrail/library_section.html", gin.H{"Error": "Erro ao carregar guardrails"})
		return
	}
	c.HTML(status, "guardrail/library_section.html", gin.H{"Guardrails": items, "Error": errMsg})
}

func (h *GuardrailLibraryHandler) RenderCreateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "guardrail/library_form.html", gin.H{"Types": guardrailTypes})
}

func (h *GuardrailLibraryHandler) RenderEditForm(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	g, err := h.repo.FindByID(c.Request.Context(), gid)
	if err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar guardrail"})
		return
	}
	c.HTML(http.StatusOK, "guardrail/library_form.html", gin.H{
		"Types": guardrailTypes,
		"Guardrail": gin.H{
			"ID": g.ID, "Name": g.Name, "Type": string(g.Type), "Rule": g.Rule, "Message": g.Message,
		},
	})
}

func (h *GuardrailLibraryHandler) HandleCreate(c *gin.Context) {
	// SpecialistID intentionally empty: library creation, unattached.
	_, err := h.createUC.Execute(c.Request.Context(), application.CreateGuardrailInput{
		Name:    c.PostForm("name"),
		Type:    c.PostForm("type"),
		Rule:    c.PostForm("rule"),
		Message: c.PostForm("message"),
	})
	if err != nil {
		respondGuardrailLibraryFormError(c, http.StatusBadRequest, mapGuardrailError(err))
		return
	}
	h.renderSection(c, http.StatusOK, "")
}

func (h *GuardrailLibraryHandler) HandleUpdate(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	_, err := h.updateUC.Execute(c.Request.Context(), application.UpdateGuardrailInput{
		ID:      gid,
		Name:    c.PostForm("name"),
		Type:    c.PostForm("type"),
		Rule:    c.PostForm("rule"),
		Message: c.PostForm("message"),
	})
	if err != nil {
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			respondGuardrailLibraryFormError(c, http.StatusNotFound, "Guardrail não encontrado.")
			return
		}
		respondGuardrailLibraryFormError(c, http.StatusBadRequest, mapGuardrailError(err))
		return
	}
	h.renderSection(c, http.StatusOK, "")
}

func (h *GuardrailLibraryHandler) HandleToggle(c *gin.Context) {
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
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ativar/desativar guardrail"})
		return
	}
	h.renderSection(c, http.StatusOK, "")
}

// HandleDelete removes a guardrail from the library. Blocked while attached to any
// specialist — the user must detach it everywhere first. Feedback re-renders the
// section with an error banner (still HTTP 200 so HTMX swaps it in).
func (h *GuardrailLibraryHandler) HandleDelete(c *gin.Context) {
	gid := c.Param("gid")
	if !validUUID(gid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	err := h.deleteUC.Execute(c.Request.Context(), gid)
	if err != nil {
		if errors.Is(err, domain.ErrGuardrailInUse) {
			count, _ := h.repo.CountSpecialistsByGuardrailID(c.Request.Context(), gid)
			h.renderSection(c, http.StatusOK, guardrailInUseMessage(count))
			return
		}
		if errors.Is(err, domain.ErrGuardrailNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Guardrail não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir guardrail"})
		return
	}
	h.renderSection(c, http.StatusOK, "")
}

func guardrailInUseMessage(count int) string {
	if count == 1 {
		return "Não é possível excluir: o guardrail está anexado a 1 especialista. Desanexe antes de excluir."
	}
	return "Não é possível excluir: o guardrail está anexado a " + strconv.Itoa(count) + " especialistas. Desanexe antes de excluir."
}

func respondGuardrailLibraryFormError(c *gin.Context, status int, message string) {
	c.Header("HX-Retarget", "#guardrail-form-error")
	c.Header("HX-Reswap", "innerHTML")
	c.HTML(status, "specialist/guardrail_form_error.html", gin.H{"Message": message})
	c.Abort()
}
