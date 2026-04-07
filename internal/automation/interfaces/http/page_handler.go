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
func (h *PageHandler) RenderFields(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderEditForm(c *gin.Context) { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleCreate(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleUpdate(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleDelete(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleToggle(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderLogs(c *gin.Context)     { c.Status(http.StatusNotImplemented) }
