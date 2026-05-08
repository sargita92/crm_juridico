package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// CrossSellRuleView is a display-friendly version of a cross-sell rule for templates.
type CrossSellRuleView struct {
	ID              string
	SpecialistID    string
	Ordem           int
	TriggerType     string
	TriggerDisplay  string // pre-formatted for display (no custom template funcs needed)
	TargetProductID string
	Ativo           bool
}

// buildTriggerDisplay formats TriggerConfig into a human-readable string.
func buildTriggerDisplay(triggerType string, cfg any) string {
	switch domain.CrossSellTriggerType(triggerType) {
	case domain.CrossSellTriggerKeyword:
		if kw, ok := cfg.(domain.KeywordTrigger); ok {
			return strings.Join(kw.Termos, ", ")
		}
	case domain.CrossSellTriggerStepAnswer:
		if sa, ok := cfg.(domain.StepAnswerTrigger); ok {
			return fmt.Sprintf("step=%s regex=%s", sa.StepID, sa.Regex)
		}
	}
	return ""
}

func ruleToView(specialistID string, r application.CrossSellRuleOutput) CrossSellRuleView {
	return CrossSellRuleView{
		ID:              r.ID,
		SpecialistID:    specialistID,
		Ordem:           r.Ordem,
		TriggerType:     r.TriggerType,
		TriggerDisplay:  buildTriggerDisplay(r.TriggerType, r.TriggerConfig),
		TargetProductID: r.TargetProductID,
		Ativo:           r.Ativo,
	}
}

// HTMXCrossSellHandler handles HTMX-specific cross-sell endpoints.
// It renders HTML fragments matching the project's lazy-load template pattern.
// B10 JSON routes (GET/POST/PUT/DELETE /cross-sell-rules) remain unchanged.
type HTMXCrossSellHandler struct {
	specialistRepo domain.SpecialistRepository
	listRulesUC    *application.ListCrossSellRulesUseCase
	createRulesUC  *application.CreateCrossSellRuleUseCase
	deleteRulesUC  *application.DeleteCrossSellRuleUseCase
	reorderRulesUC *application.ReorderCrossSellRuleUseCase
}

func NewHTMXCrossSellHandler(
	specialistRepo domain.SpecialistRepository,
	listRulesUC *application.ListCrossSellRulesUseCase,
	createRulesUC *application.CreateCrossSellRuleUseCase,
	deleteRulesUC *application.DeleteCrossSellRuleUseCase,
	reorderRulesUC *application.ReorderCrossSellRuleUseCase,
) *HTMXCrossSellHandler {
	return &HTMXCrossSellHandler{
		specialistRepo: specialistRepo,
		listRulesUC:    listRulesUC,
		createRulesUC:  createRulesUC,
		deleteRulesUC:  deleteRulesUC,
		reorderRulesUC: reorderRulesUC,
	}
}

func (h *HTMXCrossSellHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	// Section lazy-load (HTMX trigger="load")
	admin.GET("/:id/cross-sell", h.HandleSection)

	// Config toggle/update
	admin.POST("/:id/cross-sell/config", h.HandleUpdateConfig)

	// HTMX form-based rule create (form POST, not JSON)
	admin.POST("/:id/cross-sell-rules/htmx", h.HandleCreateHTMX)

	// HTMX-aware delete and reorder (return HTML section, not JSON)
	admin.DELETE("/:id/cross-sell-rules/htmx/:rule_id", h.HandleDeleteHTMX)
	admin.POST("/:id/cross-sell-rules/htmx/:rule_id/move-up", h.HandleMoveUpHTMX)
	admin.POST("/:id/cross-sell-rules/htmx/:rule_id/move-down", h.HandleMoveDownHTMX)
}

// HandleSection renders the cross-sell section fragment.
func (h *HTMXCrossSellHandler) HandleSection(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	h.renderSection(c, id, "")
}

// HandleUpdateConfig saves cross-sell config and re-renders the section.
func (h *HTMXCrossSellHandler) HandleUpdateConfig(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	spec, err := h.specialistRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar especialista"})
		return
	}

	enabled := c.PostForm("cross_sell_enabled") == "true"
	if !enabled {
		spec.DisableCrossSell()
	} else {
		mode := domain.CrossSellMode(c.PostForm("cross_sell_mode"))
		if mode == "" {
			mode = domain.CrossSellModeAnnounce
		}
		tmpl := c.PostForm("cross_sell_announcement_template")
		if err2 := spec.EnableCrossSell(mode, tmpl); err2 != nil {
			h.renderSection(c, id, mapDomainError(err2))
			return
		}
	}

	spec.AllowAICrossSellSuggestion = c.PostForm("allow_ai_cross_sell_suggestion") == "true"

	if err := h.specialistRepo.Update(c.Request.Context(), spec); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar configuracao"})
		return
	}

	h.renderSection(c, id, "")
}

// HandleCreateHTMX creates a rule from a form POST and re-renders the section.
func (h *HTMXCrossSellHandler) HandleCreateHTMX(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	triggerType := strings.TrimSpace(c.PostForm("trigger_type"))
	targetProductID := strings.TrimSpace(c.PostForm("target_product_id"))

	var triggerConfig any
	switch domain.CrossSellTriggerType(triggerType) {
	case domain.CrossSellTriggerKeyword:
		raw := c.PostForm("keywords")
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == '\n' || r == ','
		})
		var termos []string
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				termos = append(termos, t)
			}
		}
		triggerConfig = domain.KeywordTrigger{Termos: termos}

	case domain.CrossSellTriggerStepAnswer:
		triggerConfig = domain.StepAnswerTrigger{
			StepID: strings.TrimSpace(c.PostForm("step_id")),
			Regex:  strings.TrimSpace(c.PostForm("regex")),
		}

	default:
		h.renderSection(c, id, "Tipo de gatilho inválido")
		return
	}

	input := application.CreateCrossSellRuleInput{
		SpecialistID:    id,
		TriggerType:     triggerType,
		TriggerConfig:   triggerConfig,
		TargetProductID: targetProductID,
	}

	if _, err := h.createRulesUC.Execute(c.Request.Context(), input); err != nil {
		h.renderSection(c, id, mapCrossSellError(err))
		return
	}

	h.renderSection(c, id, "")
}

// HandleDeleteHTMX deletes a rule and re-renders the section as HTML.
func (h *HTMXCrossSellHandler) HandleDeleteHTMX(c *gin.Context) {
	ruleID := c.Param("rule_id")
	id := c.Param("id")
	if !validUUID(ruleID) || !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.deleteRulesUC.Execute(c.Request.Context(), ruleID); err != nil {
		if errors.Is(err, domain.ErrCrossSellRuleNotFound) {
			h.renderSection(c, id, "Regra não encontrada")
			return
		}
		h.renderSection(c, id, "Erro ao excluir regra")
		return
	}

	h.renderSection(c, id, "")
}

// HandleMoveUpHTMX moves a rule up and re-renders the section as HTML.
func (h *HTMXCrossSellHandler) HandleMoveUpHTMX(c *gin.Context) {
	ruleID := c.Param("rule_id")
	id := c.Param("id")
	if !validUUID(ruleID) || !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	err := h.reorderRulesUC.Execute(c.Request.Context(), application.ReorderCrossSellRuleInput{
		ID:        ruleID,
		Direction: "up",
	})
	if err != nil {
		if !errors.Is(err, application.ErrCrossSellRuleAlreadyFirst) {
			h.renderSection(c, id, "Erro ao mover regra")
			return
		}
	}

	h.renderSection(c, id, "")
}

// HandleMoveDownHTMX moves a rule down and re-renders the section as HTML.
func (h *HTMXCrossSellHandler) HandleMoveDownHTMX(c *gin.Context) {
	ruleID := c.Param("rule_id")
	id := c.Param("id")
	if !validUUID(ruleID) || !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	err := h.reorderRulesUC.Execute(c.Request.Context(), application.ReorderCrossSellRuleInput{
		ID:        ruleID,
		Direction: "down",
	})
	if err != nil {
		if !errors.Is(err, application.ErrCrossSellRuleAlreadyLast) {
			h.renderSection(c, id, "Erro ao mover regra")
			return
		}
	}

	h.renderSection(c, id, "")
}

// renderSection fetches specialist config + rules and renders cross_sell_section.html.
func (h *HTMXCrossSellHandler) renderSection(c *gin.Context, specialistID, errMsg string) {
	spec, err := h.specialistRepo.FindByID(c.Request.Context(), specialistID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar especialista"})
		return
	}

	rules, err := h.listRulesUC.Execute(c.Request.Context(), specialistID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar regras"})
		return
	}

	views := make([]CrossSellRuleView, len(rules))
	for i, r := range rules {
		views[i] = ruleToView(specialistID, r)
	}

	c.HTML(http.StatusOK, "specialist/cross_sell_section.html", gin.H{
		"SpecialistID":                  specialistID,
		"CrossSellEnabled":              spec.CrossSellEnabled,
		"CrossSellMode":                 string(spec.CrossSellMode),
		"CrossSellAnnouncementTemplate": spec.CrossSellAnnouncementTemplate,
		"AllowAICrossSellSuggestion":    spec.AllowAICrossSellSuggestion,
		"Rules":                         views,
		"RuleCount":                     len(rules),
		"Error":                         errMsg,
	})
}
