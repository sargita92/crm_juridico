package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// CrossSellRuleHandler handles HTTP requests for cross-sell rule CRUD.
type CrossSellRuleHandler struct {
	listUC    *application.ListCrossSellRulesUseCase
	createUC  *application.CreateCrossSellRuleUseCase
	updateUC  *application.UpdateCrossSellRuleUseCase
	deleteUC  *application.DeleteCrossSellRuleUseCase
	reorderUC *application.ReorderCrossSellRuleUseCase
}

func NewCrossSellRuleHandler(
	listUC *application.ListCrossSellRulesUseCase,
	createUC *application.CreateCrossSellRuleUseCase,
	updateUC *application.UpdateCrossSellRuleUseCase,
	deleteUC *application.DeleteCrossSellRuleUseCase,
	reorderUC *application.ReorderCrossSellRuleUseCase,
) *CrossSellRuleHandler {
	return &CrossSellRuleHandler{
		listUC:    listUC,
		createUC:  createUC,
		updateUC:  updateUC,
		deleteUC:  deleteUC,
		reorderUC: reorderUC,
	}
}

func (h *CrossSellRuleHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/cross-sell-rules", h.HandleList)
	admin.POST("/:id/cross-sell-rules", h.HandleCreate)
	admin.PUT("/:id/cross-sell-rules/:rule_id", h.HandleUpdate)
	admin.DELETE("/:id/cross-sell-rules/:rule_id", h.HandleDelete)
	admin.POST("/:id/cross-sell-rules/:rule_id/move-up", h.HandleMoveUp)
	admin.POST("/:id/cross-sell-rules/:rule_id/move-down", h.HandleMoveDown)
}

func (h *CrossSellRuleHandler) HandleList(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	items, err := h.listUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar regras de cross-sell"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rules": items})
}

func (h *CrossSellRuleHandler) HandleCreate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var body crossSellRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	triggerConfig, err := parseTriggerConfig(body.TriggerType, body.TriggerConfig)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := application.CreateCrossSellRuleInput{
		SpecialistID:    id,
		TriggerType:     body.TriggerType,
		TriggerConfig:   triggerConfig,
		TargetProductID: body.TargetProductID,
	}

	out, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapCrossSellError(err)})
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *CrossSellRuleHandler) HandleUpdate(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if !validUUID(ruleID) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var body crossSellRuleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Payload inválido"})
		return
	}

	triggerConfig, err := parseTriggerConfig(body.TriggerType, body.TriggerConfig)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input := application.UpdateCrossSellRuleInput{
		ID:              ruleID,
		TriggerType:     body.TriggerType,
		TriggerConfig:   triggerConfig,
		TargetProductID: body.TargetProductID,
		Ativo:           body.Ativo,
	}

	out, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrCrossSellRuleNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Regra não encontrada"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapCrossSellError(err)})
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *CrossSellRuleHandler) HandleDelete(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if !validUUID(ruleID) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.deleteUC.Execute(c.Request.Context(), ruleID); err != nil {
		if errors.Is(err, domain.ErrCrossSellRuleNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Regra não encontrada"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir regra"})
		return
	}

	// Re-render list after delete
	h.HandleList(c)
}

func (h *CrossSellRuleHandler) HandleMoveUp(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if !validUUID(ruleID) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	err := h.reorderUC.Execute(c.Request.Context(), application.ReorderCrossSellRuleInput{
		ID:        ruleID,
		Direction: "up",
	})
	if err != nil {
		if errors.Is(err, domain.ErrCrossSellRuleNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Regra não encontrada"})
			return
		}
		if errors.Is(err, application.ErrCrossSellRuleAlreadyFirst) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Regra já está na primeira posição"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao mover regra"})
		return
	}

	h.HandleList(c)
}

func (h *CrossSellRuleHandler) HandleMoveDown(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if !validUUID(ruleID) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	err := h.reorderUC.Execute(c.Request.Context(), application.ReorderCrossSellRuleInput{
		ID:        ruleID,
		Direction: "down",
	})
	if err != nil {
		if errors.Is(err, domain.ErrCrossSellRuleNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Regra não encontrada"})
			return
		}
		if errors.Is(err, application.ErrCrossSellRuleAlreadyLast) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Regra já está na última posição"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao mover regra"})
		return
	}

	h.HandleList(c)
}

// --- Request body ---

type crossSellRuleBody struct {
	TriggerType     string          `json:"trigger_type"`
	TriggerConfig   json.RawMessage `json:"trigger_config"`
	TargetProductID string          `json:"target_product_id"`
	Ativo           bool            `json:"ativo"`
}

// parseTriggerConfig converts raw JSON into the appropriate domain trigger config type.
func parseTriggerConfig(triggerType string, raw json.RawMessage) (any, error) {
	switch domain.CrossSellTriggerType(strings.TrimSpace(triggerType)) {
	case domain.CrossSellTriggerKeyword:
		var kw domain.KeywordTrigger
		if err := json.Unmarshal(raw, &kw); err != nil {
			return nil, domain.ErrKeywordTriggerEmpty
		}
		return kw, nil
	case domain.CrossSellTriggerStepAnswer:
		var sa domain.StepAnswerTrigger
		if err := json.Unmarshal(raw, &sa); err != nil {
			return nil, domain.ErrStepAnswerTriggerInvalid
		}
		return sa, nil
	default:
		return nil, domain.ErrUnsupportedTrigger
	}
}

// mapCrossSellError maps domain errors to user-facing messages.
func mapCrossSellError(err error) string {
	switch {
	case errors.Is(err, domain.ErrTargetProductRequired):
		return "Produto alvo é obrigatório"
	case errors.Is(err, domain.ErrKeywordTriggerEmpty):
		return "Trigger de palavras-chave deve ter ao menos um termo"
	case errors.Is(err, domain.ErrStepAnswerTriggerInvalid):
		return "Trigger de resposta de step requer um step ID válido"
	case errors.Is(err, domain.ErrInvalidRegex):
		return "Regex do trigger é inválida"
	case errors.Is(err, domain.ErrUnsupportedTrigger):
		return "Tipo de trigger não suportado"
	case errors.Is(err, domain.ErrSpecialistIDRequired):
		return "ID do especialista é obrigatório"
	default:
		return "Erro interno"
	}
}
