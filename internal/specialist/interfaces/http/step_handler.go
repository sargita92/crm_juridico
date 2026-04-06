package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type StepHandler struct {
	createUC *application.CreateStepUseCase
	updateUC *application.UpdateStepUseCase
	deleteUC *application.DeleteStepUseCase
	moveUC   *application.MoveStepUseCase
	listUC   *application.ListStepsUseCase
}

func NewStepHandler(
	createUC *application.CreateStepUseCase,
	updateUC *application.UpdateStepUseCase,
	deleteUC *application.DeleteStepUseCase,
	moveUC *application.MoveStepUseCase,
	listUC *application.ListStepsUseCase,
) *StepHandler {
	return &StepHandler{
		createUC: createUC, updateUC: updateUC,
		deleteUC: deleteUC, moveUC: moveUC, listUC: listUC,
	}
}

func (h *StepHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/steps", h.HandleList)
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

func (h *StepHandler) HandleCreate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	score, err := parseScore(c.DefaultPostForm("score", "0"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Pontuação inválida"})
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
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, domain.ErrSpecialistInactive) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapStepError(err)})
		return
	}

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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Pontuação inválida"})
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
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Step não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapStepError(err)})
		return
	}

	h.HandleList(c)
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
