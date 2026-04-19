package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/automation/application"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// PageHandler serves HTML pages for the automation management screens.
type PageHandler struct {
	crudUC          *application.CRUDUseCase
	listFunnelsUC   *funnelapp.ListFunnelsUseCase
	columnRepo      funneldomain.ColumnRepository
	leadRepo        funneldomain.LeadRepository
	contactProvider funneldomain.ContactProvider
	specialistRepo  specialistdomain.SpecialistRepository
	specTenantRepo  specialistdomain.SpecialistTenantRepository
	log             *zap.Logger
}

// NewPageHandler constructs a PageHandler.
func NewPageHandler(
	crudUC *application.CRUDUseCase,
	listFunnelsUC *funnelapp.ListFunnelsUseCase,
	columnRepo funneldomain.ColumnRepository,
	leadRepo funneldomain.LeadRepository,
	contactProvider funneldomain.ContactProvider,
	specialistRepo specialistdomain.SpecialistRepository,
	specTenantRepo specialistdomain.SpecialistTenantRepository,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{
		crudUC:          crudUC,
		listFunnelsUC:   listFunnelsUC,
		columnRepo:      columnRepo,
		leadRepo:        leadRepo,
		contactProvider: contactProvider,
		specialistRepo:  specialistRepo,
		specTenantRepo:  specTenantRepo,
		log:             log,
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

// --- Page Handlers ---

// ListPage handles GET /tenant/automations — renders the full page.
func (h *PageHandler) ListPage(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.list")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
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
	})
}

// RenderTable handles GET /tenant/automations/table — returns table fragment.
func (h *PageHandler) RenderTable(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.table")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	if funnelID == "" {
		funnelID = c.PostForm("funnel_id")
	}

	automations, columns := h.loadTableData(c, tenantID, funnelID)

	c.HTML(http.StatusOK, "automation/table.html", gin.H{
		"Automations": h.enrichAutomations(automations, columns),
	})
}

// RenderCreateForm handles GET /tenant/automations/new/form — returns modal form for creation.
func (h *PageHandler) RenderCreateForm(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.create_form")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	if funnelID == "" {
		funnelID = c.PostForm("funnel_id")
	}

	// Load funnels to get the funnel_id from the page if not provided
	if funnelID == "" {
		funnels, _ := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
		if len(funnels) > 0 {
			funnelID = funnels[0].ID
		}
	}

	columns, _ := h.columnRepo.FindByFunnelID(c.Request.Context(), funnelID)

	c.HTML(http.StatusOK, "automation/modal_form.html", gin.H{
		"CurrentFunnelID":  funnelID,
		"Columns":          columns,
		"SelectedType":     "expiration",
		"SelectedColumnID": "",
		"SelectedPriority": 0,
	})
}

// --- Table helpers ---

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

// --- Dynamic Fields ---

var validFieldTypes = map[string]bool{
	"expiration": true, "move_funnel": true, "auto_message": true,
	"auto_note": true, "switch_specialist": true, "rate_limit": true,
	"detect_product": true,
}

func (h *PageHandler) RenderFields(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.fields")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())
	automationType := c.Query("type")
	if !validFieldTypes[automationType] {
		c.String(http.StatusBadRequest, "Tipo inválido")
		return
	}
	automationID := c.Query("automation_id")

	data := gin.H{}

	if automationID != "" {
		auto, err := h.crudUC.GetByID(c.Request.Context(), automationID)
		if err == nil {
			h.addConfigData(data, automationType, auto.Config, c, tenantID)
		}
	}

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

	c.HTML(http.StatusOK, "automation/fields_"+automationType+".html", data)
}

type specialistOption struct {
	ID   string
	Name string
}

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

// --- Edit Form ---

func (h *PageHandler) RenderEditForm(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.edit_form")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
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

// --- Create / Update ---

func (h *PageHandler) HandleCreate(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.handle_create")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	if funnelID == "" {
		funnelID = c.PostForm("funnel_id")
	}

	automationType := c.PostForm("type")
	columnID := c.PostForm("column_id")
	priority, _ := strconv.Atoi(c.PostForm("priority"))
	config := buildConfig(automationType, c.Request.PostForm)

	_, err := h.crudUC.Create(c.Request.Context(), application.CreateAutomationInput{
		TenantID: tenantID,
		FunnelID: funnelID,
		ColumnID: columnID,
		Type:     automationType,
		Config:   config,
		Priority: priority,
	})
	if err != nil {
		h.log.Error("failed to create automation", zap.Error(err))
		columns, _ := h.columnRepo.FindByFunnelID(c.Request.Context(), funnelID)
		data := gin.H{
			"FormError":        err.Error(),
			"CurrentFunnelID":  funnelID,
			"Columns":          columns,
			"SelectedColumnID": columnID,
			"SelectedType":     automationType,
			"SelectedPriority": priority,
		}
		h.addConfigData(data, automationType, config, c, tenantID)
		c.HTML(http.StatusOK, "automation/modal_form.html", data)
		return
	}

	c.Header("HX-Trigger", "refreshTable")
	c.String(http.StatusOK, "")
}

func (h *PageHandler) HandleUpdate(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.handle_update")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	tenantID := middleware.GetTenantID(c.Request.Context())
	id := c.Param("id")

	automationType := c.PostForm("type")
	columnID := c.PostForm("column_id")
	funnelID := c.PostForm("funnel_id")
	priority, _ := strconv.Atoi(c.PostForm("priority"))
	config := buildConfig(automationType, c.Request.PostForm)

	_, err := h.crudUC.Update(c.Request.Context(), id, application.CreateAutomationInput{
		ColumnID: columnID,
		Config:   config,
		Priority: priority,
	})
	if err != nil {
		h.log.Error("failed to update automation", zap.String("id", id), zap.Error(err))
		auto, _ := h.crudUC.GetByID(c.Request.Context(), id)
		columns, _ := h.columnRepo.FindByFunnelID(c.Request.Context(), funnelID)
		data := gin.H{
			"FormError":        err.Error(),
			"IsEdit":           true,
			"Automation":       auto,
			"CurrentFunnelID":  funnelID,
			"Columns":          columns,
			"SelectedColumnID": columnID,
			"SelectedType":     automationType,
			"SelectedPriority": priority,
		}
		h.addConfigData(data, automationType, config, c, tenantID)
		c.HTML(http.StatusOK, "automation/modal_form.html", data)
		return
	}

	c.Header("HX-Trigger", "refreshTable")
	c.String(http.StatusOK, "")
}

// --- Delete / Toggle ---

func (h *PageHandler) HandleDelete(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.handle_delete")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	id := c.Param("id")

	if err := h.crudUC.Delete(c.Request.Context(), id); err != nil {
		h.log.Error("failed to delete automation", zap.String("id", id), zap.Error(err))
		c.String(http.StatusNotFound, "Automação não encontrada")
		return
	}

	h.RenderTable(c)
}

func (h *PageHandler) HandleToggle(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.handle_toggle")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	id := c.Param("id")

	if err := h.crudUC.Toggle(c.Request.Context(), id); err != nil {
		h.log.Error("failed to toggle automation", zap.String("id", id), zap.Error(err))
		c.String(http.StatusNotFound, "Automação não encontrada")
		return
	}

	h.RenderTable(c)
}

// --- Logs ---

// logView enriches LogOutput with the contact name for display.
type logView struct {
	LeadName     string
	Status       string
	ErrorMessage string
	ExecutedAt   interface{} // time.Time — passed through for template .Format
}

func (h *PageHandler) RenderLogs(c *gin.Context) {
	ctx, span := otel.Tracer("automation").Start(c.Request.Context(), "automation.page.logs")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)
	id := c.Param("id")

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	logs, err := h.crudUC.GetLogs(c.Request.Context(), id, limit+1, offset)
	if err != nil {
		h.log.Error("failed to get logs", zap.String("id", id), zap.Error(err))
		c.String(http.StatusInternalServerError, "Erro ao carregar logs")
		return
	}

	hasNext := len(logs) > limit
	if hasNext {
		logs = logs[:limit]
	}

	// Enrich logs with contact names
	views := make([]logView, len(logs))
	for i, l := range logs {
		name := h.resolveLeadName(c, l.LeadID)
		views[i] = logView{
			LeadName:     name,
			Status:       l.Status,
			ErrorMessage: l.ErrorMessage,
			ExecutedAt:   l.ExecutedAt,
		}
	}

	c.HTML(http.StatusOK, "automation/modal_logs.html", gin.H{
		"Logs":         views,
		"AutomationID": id,
		"Limit":        limit,
		"StartItem":    offset + 1,
		"EndItem":      offset + len(views),
		"HasPrev":      offset > 0,
		"HasNext":      hasNext,
		"PrevOffset":   offset - limit,
		"NextOffset":   offset + limit,
	})
}

// resolveLeadName looks up Lead → Contact to get the contact name.
func (h *PageHandler) resolveLeadName(c *gin.Context, leadID string) string {
	lead, err := h.leadRepo.FindByID(c.Request.Context(), leadID)
	if err != nil {
		return leadID[:8] + "…"
	}

	if lead.ContactID != "" && h.contactProvider != nil {
		info, err := h.contactProvider.FindByID(c.Request.Context(), lead.ContactID)
		if err == nil && info.Name != "" {
			return info.Name
		}
	}

	return leadID[:8] + "…"
}
