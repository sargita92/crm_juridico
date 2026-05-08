package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ScoringHandler struct {
	getUC    *application.GetScoringUseCase
	updateUC *application.UpdateScoringUseCase
}

func NewScoringHandler(
	getUC *application.GetScoringUseCase,
	updateUC *application.UpdateScoringUseCase,
) *ScoringHandler {
	return &ScoringHandler{getUC: getUC, updateUC: updateUC}
}

func (h *ScoringHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/scoring", h.HandleGet)
	admin.PUT("/:id/scoring", h.HandleUpdate)
}

func (h *ScoringHandler) HandleGet(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar scoring"})
		return
	}
	c.HTML(http.StatusOK, "specialist/scoring_section.html", gin.H{
		"Scoring":      output,
		"SpecialistID": id,
	})
}

func (h *ScoringHandler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	threshold, err := strconv.Atoi(c.PostForm("threshold"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold inválido"})
		return
	}

	thresholdHumanoMinStr := c.PostForm("threshold_humano_min")
	thresholdHumanoMin := 0
	if thresholdHumanoMinStr != "" {
		thresholdHumanoMin, err = strconv.Atoi(thresholdHumanoMinStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold humano min inválido"})
			return
		}
	}

	_, err = h.updateUC.Execute(c.Request.Context(), application.UpdateScoringInput{
		SpecialistID:         id,
		Threshold:            threshold,
		ThresholdHumanoMin:   thresholdHumanoMin,
		QualifiedColumnID:    c.PostForm("qualified_column_id"),
		HumanColumnID:        c.PostForm("human_column_id"),
		DisqualifiedColumnID: c.PostForm("disqualified_column_id"),
		CrossSellColumnID:    c.PostForm("cross_sell_column_id"),
	})
	if err != nil {
		if errors.Is(err, domain.ErrScoringThresholdInvalid) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold deve ser maior que zero"})
			return
		}
		if errors.Is(err, domain.ErrScoringThresholdExceedsTotal) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold não pode exceder o total possível"})
			return
		}
		if errors.Is(err, domain.ErrHumanoMinAboveAprovado) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold humano min não pode ser maior que o threshold aprovado"})
			return
		}
		if errors.Is(err, domain.ErrHumanoMinNegative) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Threshold humano min não pode ser negativo"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao atualizar scoring"})
		return
	}

	h.HandleGet(c)
}
