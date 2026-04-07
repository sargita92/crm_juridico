package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/automation/application"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// PageHandler serves HTML pages for the automation management screens.
type PageHandler struct {
	crudUC         *application.CRUDUseCase
	listFunnelsUC  *funnelapp.ListFunnelsUseCase
	columnRepo     funneldomain.ColumnRepository
	specialistRepo specialistdomain.SpecialistRepository
	specTenantRepo specialistdomain.SpecialistTenantRepository
	log            *zap.Logger
}

// NewPageHandler constructs a PageHandler.
func NewPageHandler(
	crudUC *application.CRUDUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	specialistRepo specialistdomain.SpecialistRepository,
	specTenantRepo specialistdomain.SpecialistTenantRepository,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		crudUC:         crudUC,
		listFunnelsUC:  listFunnelsUC,
		columnRepo:     columnRepo,
		specialistRepo: specialistRepo,
		specTenantRepo: specTenantRepo,
		log:            log,
	}
}

// typeLabel maps automation type to a human-readable label with icon.
var typeLabel = map[string]string{
	"expiration":        "⏰ Exclusão por tempo",
	"move_funnel":       "🔀 Mover para funil",
	"auto_message":      "💬 Mensagem automática",
	"auto_note":         "📋 Anotação automática",
	"switch_specialist": "👤 Trocar especialista",
	"rate_limit":        "🚫 Limite de mensagens",
	"detect_product":    "🔍 Detectar produto",
}

// configSummary returns a short human-readable description of the config.
func configSummary(automationType string, config map[string]interface{}) string {
	str := func(key string) string {
		v, _ := config[key].(string)
		return v
	}
	num := func(key string) float64 {
		v, _ := config[key].(float64)
		return v
	}

	switch automationType {
	case "expiration":
		action := str("action")
		hours := num("duration_hours")
		if action == "delete" {
			return fmt.Sprintf("Excluir após %.0fh", hours)
		}
		return fmt.Sprintf("Arquivar após %.0fh", hours)
	case "move_funnel":
		return "→ " + str("target_funnel_id")
	case "auto_message", "auto_note":
		tmpl := str("template")
		runes := []rune(tmpl)
		if len(runes) > 40 {
			tmpl = string(runes[:40]) + "..."
		}
		return tmpl
	case "switch_specialist":
		return "Especialista: " + str("specialist_id")
	case "rate_limit":
		return fmt.Sprintf("Max %.0f msg / %.0fh", num("max_messages"), num("period_hours"))
	case "detect_product":
		return "Detectar produto e redirecionar"
	default:
		return ""
	}
}

// buildConfig parses form values into the config map for the given automation type.
func buildConfig(automationType string, form url.Values) map[string]interface{} {
	cfg := map[string]interface{}{}

	switch automationType {
	case "expiration":
		cfg["action"] = form.Get("config_action")
		if v, err := strconv.ParseFloat(form.Get("config_duration_hours"), 64); err == nil {
			cfg["duration_hours"] = v
		}
	case "move_funnel":
		cfg["target_funnel_id"] = form.Get("config_target_funnel_id")
		cfg["target_column_id"] = form.Get("config_target_column_id")
	case "auto_message":
		cfg["template"] = form.Get("config_template")
	case "auto_note":
		cfg["template"] = form.Get("config_template")
	case "switch_specialist":
		cfg["specialist_id"] = form.Get("config_specialist_id")
	case "rate_limit":
		if v, err := strconv.ParseFloat(form.Get("config_max_messages"), 64); err == nil {
			cfg["max_messages"] = v
		}
		if v, err := strconv.ParseFloat(form.Get("config_period_hours"), 64); err == nil {
			cfg["period_hours"] = v
		}
	case "detect_product":
		cfg["switch_specialist"] = form.Get("config_switch_specialist") == "true"
	}

	return cfg
}

// ListPage handles GET /tenant/automations — renders the full page.
func (h *PageHandler) ListPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "automation/list.html", gin.H{
			"ActiveNav": "automations",
			"Error":     "Erro ao carregar funis",
		})
		return
	}

	var currentFunnelID string
	if qf := c.Query("funnel_id"); qf != "" {
		currentFunnelID = qf
	} else if len(funnels) > 0 {
		currentFunnelID = funnels[0].ID
	}

	automations, columns := h.loadTableData(c, tenantID, currentFunnelID)

	c.HTML(http.StatusOK, "automation/list.html", gin.H{
		"ActiveNav":       "automations",
		"Funnels":         funnels,
		"CurrentFunnelID": currentFunnelID,
		"Automations":     h.enrichAutomations(automations, columns),
		"Columns":         columns,
	})
}

// RenderTable handles GET /tenant/automations/table — returns table fragment.
func (h *PageHandler) RenderTable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")

	automations, columns := h.loadTableData(c, tenantID, funnelID)

	c.HTML(http.StatusOK, "automation/table.html", gin.H{
		"Automations": h.enrichAutomations(automations, columns),
	})
}

// loadTableData fetches automations and columns for a funnel.
func (h *PageHandler) loadTableData(c *gin.Context, tenantID, funnelID string) ([]application.AutomationOutput, []funneldomain.Column) {
	var automations []application.AutomationOutput
	var columns []funneldomain.Column

	if funnelID != "" {
		var err error
		automations, err = h.crudUC.ListByFunnel(c.Request.Context(), tenantID, funnelID)
		if err != nil {
			h.log.Error("failed to list automations", zap.Error(err))
		}
		columns, err = h.columnRepo.FindByFunnelID(c.Request.Context(), funnelID)
		if err != nil {
			h.log.Error("failed to list columns", zap.Error(err))
		}
	}

	return automations, columns
}

// automationView holds enriched data for template rendering.
type automationView struct {
	ID            string
	FunnelID      string
	ColumnID      string
	Type          string
	TypeLabel     string
	ColumnName    string
	Config        map[string]interface{}
	ConfigSummary string
	Active        bool
	Priority      int
}

// enrichAutomations builds template-ready views from AutomationOutput + columns.
func (h *PageHandler) enrichAutomations(automations []application.AutomationOutput, columns []funneldomain.Column) []automationView {
	colMap := make(map[string]string, len(columns))
	for _, col := range columns {
		colMap[col.ID] = col.Name
	}

	views := make([]automationView, len(automations))
	for i, a := range automations {
		views[i] = automationView{
			ID:            a.ID,
			FunnelID:      a.FunnelID,
			ColumnID:      a.ColumnID,
			Type:          a.Type,
			TypeLabel:     typeLabel[a.Type],
			ColumnName:    colMap[a.ColumnID],
			Config:        a.Config,
			ConfigSummary: configSummary(a.Type, a.Config),
			Active:        a.Active,
			Priority:      a.Priority,
		}
	}
	return views
}
// RenderFields handles GET /tenant/automations/fields — returns dynamic fields partial.
func (h *PageHandler) RenderFields(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	automationType := c.Query("type")
	automationID := c.Query("automation_id")

	data := gin.H{}

	// Pre-populate config values when editing an existing automation
	if automationID != "" {
		auto, err := h.crudUC.GetByID(c.Request.Context(), automationID)
		if err == nil {
			h.addConfigData(data, automationType, auto.Config, c, tenantID)
		}
	}

	// Add type-specific dropdown data (funnels, specialists)
	switch automationType {
	case "move_funnel":
		if _, ok := data["Funnels"]; !ok {
			funnels, _ := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
			data["Funnels"] = funnels
		}
	case "switch_specialist":
		if _, ok := data["Specialists"]; !ok {
			data["Specialists"] = h.loadSpecialists(c, tenantID)
		}
	}

	templateName := "automation/fields_" + automationType + ".html"
	c.HTML(http.StatusOK, templateName, data)
}

// specialistOption is a simplified specialist view for select dropdowns.
type specialistOption struct {
	ID   string
	Name string
}

// loadSpecialists returns specialists available for the tenant.
func (h *PageHandler) loadSpecialists(c *gin.Context, tenantID string) []specialistOption {
	specIDs, err := h.specTenantRepo.FindSpecialistIDsByTenantID(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to load specialist IDs", zap.Error(err))
		return nil
	}

	var options []specialistOption
	for _, id := range specIDs {
		spec, err := h.specialistRepo.FindByID(c.Request.Context(), id)
		if err != nil {
			continue
		}
		options = append(options, specialistOption{ID: spec.ID, Name: spec.Name})
	}
	return options
}

// RenderEditForm handles GET /tenant/automations/:id/form — returns modal pre-filled for editing.
func (h *PageHandler) RenderEditForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	auto, err := h.crudUC.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get automation", zap.String("id", id), zap.Error(err))
		c.String(http.StatusNotFound, "Automação não encontrada")
		return
	}

	columns, _ := h.columnRepo.FindByFunnelID(c.Request.Context(), auto.FunnelID)

	data := gin.H{
		"IsEdit":           true,
		"Automation":       auto,
		"CurrentFunnelID":  auto.FunnelID,
		"Columns":          columns,
		"SelectedColumnID": auto.ColumnID,
		"SelectedType":     auto.Type,
		"SelectedPriority": auto.Priority,
	}

	h.addConfigData(data, auto.Type, auto.Config, c, tenantID)

	c.HTML(http.StatusOK, "automation/modal_form.html", data)
}

// addConfigData adds type-specific config values to the template data.
func (h *PageHandler) addConfigData(data gin.H, automationType string, config map[string]interface{}, c *gin.Context, tenantID string) {
	str := func(key string) string {
		v, _ := config[key].(string)
		return v
	}
	num := func(key string) float64 {
		v, _ := config[key].(float64)
		return v
	}
	boolean := func(key string) bool {
		v, _ := config[key].(bool)
		return v
	}

	switch automationType {
	case "expiration":
		data["ConfigAction"] = str("action")
		data["ConfigDurationHours"] = num("duration_hours")
	case "move_funnel":
		data["ConfigTargetFunnelID"] = str("target_funnel_id")
		data["ConfigTargetColumnID"] = str("target_column_id")
		funnels, _ := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
		data["Funnels"] = funnels
	case "auto_message", "auto_note":
		data["ConfigTemplate"] = str("template")
	case "switch_specialist":
		data["ConfigSpecialistID"] = str("specialist_id")
		data["Specialists"] = h.loadSpecialists(c, tenantID)
	case "rate_limit":
		data["ConfigMaxMessages"] = num("max_messages")
		data["ConfigPeriodHours"] = num("period_hours")
	case "detect_product":
		data["ConfigSwitchSpecialist"] = boolean("switch_specialist")
	}
}
func (h *PageHandler) HandleCreate(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleUpdate(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
// HandleDelete handles DELETE /tenant/automations/:id — deletes and returns table.
func (h *PageHandler) HandleDelete(c *gin.Context) {
	id := c.Param("id")

	if err := h.crudUC.Delete(c.Request.Context(), id); err != nil {
		h.log.Error("failed to delete automation", zap.String("id", id), zap.Error(err))
		c.String(http.StatusNotFound, "Automação não encontrada")
		return
	}

	h.RenderTable(c)
}

// HandleToggle handles POST /tenant/automations/:id/toggle — toggles and returns table.
func (h *PageHandler) HandleToggle(c *gin.Context) {
	id := c.Param("id")

	if err := h.crudUC.Toggle(c.Request.Context(), id); err != nil {
		h.log.Error("failed to toggle automation", zap.String("id", id), zap.Error(err))
		c.String(http.StatusNotFound, "Automação não encontrada")
		return
	}

	h.RenderTable(c)
}
func (h *PageHandler) RenderLogs(c *gin.Context)     { c.Status(http.StatusNotImplemented) }
