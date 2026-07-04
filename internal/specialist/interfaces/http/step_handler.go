package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// stepDataTypes é a lista canônica de opções exibidas no select Tipo de dado.
// Mantida aqui para evitar duplicação entre o template (que apenas itera) e o
// backend (que precisa rejeitar valores fora do conjunto).
var stepDataTypes = []struct{ Value, Label string }{
	{"free_text", "Texto livre"},
	{"number", "Número"},
	{"date", "Data"},
	{"document", "Documento"},
	{"selection", "Seleção"},
}

type StepHandler struct {
	createUC *application.CreateStepUseCase
	updateUC *application.UpdateStepUseCase
	deleteUC *application.DeleteStepUseCase
	moveUC   *application.MoveStepUseCase
	listUC   *application.ListStepsUseCase
	stepRepo domain.StepRepository
}

func NewStepHandler(
	createUC *application.CreateStepUseCase,
	updateUC *application.UpdateStepUseCase,
	deleteUC *application.DeleteStepUseCase,
	moveUC *application.MoveStepUseCase,
	listUC *application.ListStepsUseCase,
	stepRepo domain.StepRepository,
) *StepHandler {
	return &StepHandler{
		createUC: createUC, updateUC: updateUC,
		deleteUC: deleteUC, moveUC: moveUC, listUC: listUC,
		stepRepo: stepRepo,
	}
}

func (h *StepHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/steps", h.HandleList)
	admin.GET("/:id/steps/new/form", h.RenderCreateForm)
	admin.GET("/:id/steps/:sid/form", h.RenderEditForm)
	admin.POST("/:id/steps", h.HandleCreate)
	admin.PUT("/:id/steps/:sid", h.HandleUpdate)
	admin.POST("/:id/steps/:sid/move-up", h.HandleMoveUp)
	admin.POST("/:id/steps/:sid/move-down", h.HandleMoveDown)
	admin.DELETE("/:id/steps/:sid", h.HandleDelete)
}

func (h *StepHandler) HandleList(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	items, err := h.listUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar steps"})
		return
	}
	c.HTML(http.StatusOK, "specialist/steps_section.html", gin.H{
		"Steps":        items,
		"SpecialistID": id,
		"StepCount":    len(items),
	})
}

// RenderCreateForm responde GET /admin/specialists/:id/steps/new/form com o
// partial step_form.html vazio, consumido pelo #step-modal-body via HTMX.
func (h *StepHandler) RenderCreateForm(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	c.HTML(http.StatusOK, "specialist/step_form.html", gin.H{
		"SpecialistID": id,
		"DataTypes":    stepDataTypes,
	})
}

// RenderEditForm responde GET /admin/specialists/:id/steps/:sid/form com o
// partial step_form.html pré-preenchido pelo step persistido. Antes só era
// possível editar excluindo e recriando.
func (h *StepHandler) RenderEditForm(c *gin.Context) {
	id := c.Param("id")
	sid := c.Param("sid")
	if !validUUID(id) || !validUUID(sid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	step, err := h.stepRepo.FindByID(c.Request.Context(), sid)
	if err != nil {
		if errors.Is(err, domain.ErrStepNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Step não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar step"})
		return
	}
	c.HTML(http.StatusOK, "specialist/step_form.html", gin.H{
		"SpecialistID": id,
		"DataTypes":    stepDataTypes,
		"Step": gin.H{
			"ID":       step.ID,
			"Text":     step.Text,
			"DataType": string(step.DataType),
			"Required": step.Required,
			"Score":    step.Score,
		},
	})
}

func (h *StepHandler) HandleCreate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	score, err := parseScore(c.DefaultPostForm("score", "0"))
	if err != nil {
		respondStepFormError(c, http.StatusBadRequest, "Pontuação inválida. Use um número inteiro positivo.")
		return
	}
	required := c.PostForm("required") == "on" || c.PostForm("required") == "true"

	input := application.CreateStepInput{
		SpecialistID: id,
		Text:         c.PostForm("text"),
		DataType:     c.PostForm("data_type"),
		Required:     required,
		Score:        score,
	}

	_, err = h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			respondStepFormError(c, http.StatusNotFound, "Especialista não encontrado.")
			return
		}
		if errors.Is(err, domain.ErrSpecialistInactive) {
			respondStepFormError(c, http.StatusBadRequest, "Especialista está inativo. Reative antes de adicionar steps.")
			return
		}
		respondStepFormError(c, http.StatusBadRequest, mapStepError(err))
		return
	}

	c.Header("HX-Trigger", buildStepToast("Step adicionado com sucesso.", "success"))
	h.HandleList(c)
}

func (h *StepHandler) HandleUpdate(c *gin.Context) {
	sid := c.Param("sid")
	if !validUUID(sid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	score, err := parseScore(c.DefaultPostForm("score", "0"))
	if err != nil {
		respondStepFormError(c, http.StatusBadRequest, "Pontuação inválida. Use um número inteiro positivo.")
		return
	}
	required := c.PostForm("required") == "on" || c.PostForm("required") == "true"

	input := application.UpdateStepInput{
		ID:       sid,
		Text:     c.PostForm("text"),
		DataType: c.PostForm("data_type"),
		Required: required,
		Score:    score,
	}

	_, err = h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrStepNotFound) {
			respondStepFormError(c, http.StatusNotFound, "Step não encontrado.")
			return
		}
		respondStepFormError(c, http.StatusBadRequest, mapStepError(err))
		return
	}

	c.Header("HX-Trigger", buildStepToast("Step atualizado com sucesso.", "success"))
	h.HandleList(c)
}

// respondStepFormError redireciona o alerta HTML para o slot #step-form-error
// dentro do modal — antes o JSON 400 ia para o #steps-section (hx-target do
// form) e corrompia a lista de steps. Mesmo padrão usado em tenants_modal.
func respondStepFormError(c *gin.Context, status int, message string) {
	c.Header("HX-Retarget", "#step-form-error")
	c.Header("HX-Reswap", "innerHTML")
	c.HTML(status, "specialist/step_form_error.html", gin.H{"Message": message})
	c.Abort()
}

// buildStepToast monta o payload do HX-Trigger consumido por admin.js -> showAdminToast.
// QuoteToASCII escapa não-ASCII: HTTP header é ISO-8859-1 e mangleia UTF-8 cru.
func buildStepToast(message, kind string) string {
	return fmt.Sprintf(`{"adminToast":{"message":%s,"kind":%s}}`, strconv.QuoteToASCII(message), strconv.QuoteToASCII(kind))
}

func (h *StepHandler) HandleMoveUp(c *gin.Context) {
	sid := c.Param("sid")
	if !validUUID(sid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	err := h.moveUC.Execute(c.Request.Context(), application.MoveStepInput{ID: sid, Direction: "up"})
	if err != nil {
		if errors.Is(err, domain.ErrStepNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Step não encontrado"})
			return
		}
		if errors.Is(err, application.ErrStepAlreadyFirst) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Step já está na primeira posição"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao mover step"})
		return
	}

	h.HandleList(c)
}

func (h *StepHandler) HandleMoveDown(c *gin.Context) {
	sid := c.Param("sid")
	if !validUUID(sid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	err := h.moveUC.Execute(c.Request.Context(), application.MoveStepInput{ID: sid, Direction: "down"})
	if err != nil {
		if errors.Is(err, domain.ErrStepNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Step não encontrado"})
			return
		}
		if errors.Is(err, application.ErrStepAlreadyLast) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Step já está na última posição"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao mover step"})
		return
	}

	h.HandleList(c)
}

func (h *StepHandler) HandleDelete(c *gin.Context) {
	sid := c.Param("sid")
	if !validUUID(sid) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	if err := h.deleteUC.Execute(c.Request.Context(), sid); err != nil {
		if errors.Is(err, domain.ErrStepNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Step não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir step"})
		return
	}

	h.HandleList(c)
}

func parseScore(s string) (int, error) {
	score, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if score < 0 {
		return 0, domain.ErrStepScoreNegative
	}
	return score, nil
}

func mapStepError(err error) string {
	switch {
	case errors.Is(err, domain.ErrStepTextRequired):
		return "Texto é obrigatório"
	case errors.Is(err, domain.ErrStepTextTooLong):
		return "Texto excede o tamanho máximo"
	case errors.Is(err, domain.ErrStepDataTypeInvalid):
		return "Tipo de dado inválido"
	case errors.Is(err, domain.ErrStepScoreNegative):
		return "Pontuação não pode ser negativa"
	default:
		return "Erro interno"
	}
}
