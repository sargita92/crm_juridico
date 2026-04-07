# F09 Step 7 — Telas de Automações (HTMX) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build HTMX screens for managing automations — list with funnel filter, inline toggle, modal CRUD with dynamic fields per type, and execution logs modal.

**Architecture:** New `PageHandler` in the automation module renders HTML templates via Gin's `c.HTML()`. It reuses the existing `CRUDUseCase` for all operations and receives cross-module dependencies (funnels, columns, specialists) for populating dropdowns. Form submissions return `HX-Trigger` headers to refresh the table and close modals.

**Tech Stack:** Go, Gin, HTMX 2.0.4, Go html/template, existing CSS from main.css

**Spec:** `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-step7-screens.md`

---

## File Structure

```
Create: internal/automation/interfaces/http/page_handler.go      — HTML page handler (all rendering + form handling)
Create: internal/automation/interfaces/http/page_handler_test.go  — Handler tests (config building, HTTP responses)
Modify: internal/automation/interfaces/http/routes.go             — Register HTML routes
Modify: internal/automation/module.go                             — Wire PageHandler with cross-module deps
Modify: cmd/api/main.go                                          — Pass new deps to automation module
Modify: web/templates/partials/tenant_sidebar.html                — Add "Automações" nav item
Create: web/templates/automation/list.html                        — Full page (extends tenant layout)
Create: web/templates/automation/table.html                       — Table fragment (HTMX-swappable)
Create: web/templates/automation/modal_form.html                  — Create/edit modal
Create: web/templates/automation/modal_logs.html                  — Execution logs modal
Create: web/templates/automation/fields_expiration.html           — Dynamic fields: expiration
Create: web/templates/automation/fields_move_funnel.html          — Dynamic fields: move funnel
Create: web/templates/automation/fields_auto_message.html         — Dynamic fields: auto message
Create: web/templates/automation/fields_auto_note.html            — Dynamic fields: auto note
Create: web/templates/automation/fields_switch_specialist.html    — Dynamic fields: switch specialist
Create: web/templates/automation/fields_rate_limit.html           — Dynamic fields: rate limit
Create: web/templates/automation/fields_detect_product.html       — Dynamic fields: detect product
```

---

### Task 1: PageHandler scaffold + module wiring

**Files:**
- Create: `internal/automation/interfaces/http/page_handler.go`
- Create: `internal/automation/interfaces/http/page_handler_test.go`
- Modify: `internal/automation/interfaces/http/routes.go`
- Modify: `internal/automation/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Write test for buildConfig helper**

```go
// internal/automation/interfaces/http/page_handler_test.go
package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConfig_Expiration(t *testing.T) {
	form := url.Values{
		"config_action":         {"archive"},
		"config_duration_hours": {"48"},
	}
	cfg := buildConfig("expiration", form)
	assert.Equal(t, "archive", cfg["action"])
	assert.Equal(t, float64(48), cfg["duration_hours"])
}

func TestBuildConfig_MoveFunnel(t *testing.T) {
	form := url.Values{
		"config_target_funnel_id": {"funnel-123"},
		"config_target_column_id": {"col-456"},
	}
	cfg := buildConfig("move_funnel", form)
	assert.Equal(t, "funnel-123", cfg["target_funnel_id"])
	assert.Equal(t, "col-456", cfg["target_column_id"])
}

func TestBuildConfig_AutoMessage(t *testing.T) {
	form := url.Values{
		"config_template": {"Olá {{nome}}, recebemos sua mensagem"},
	}
	cfg := buildConfig("auto_message", form)
	assert.Equal(t, "Olá {{nome}}, recebemos sua mensagem", cfg["template"])
}

func TestBuildConfig_AutoNote(t *testing.T) {
	form := url.Values{
		"config_template": {"Lead qualificado automaticamente"},
	}
	cfg := buildConfig("auto_note", form)
	assert.Equal(t, "Lead qualificado automaticamente", cfg["template"])
}

func TestBuildConfig_SwitchSpecialist(t *testing.T) {
	form := url.Values{
		"config_specialist_id": {"spec-789"},
	}
	cfg := buildConfig("switch_specialist", form)
	assert.Equal(t, "spec-789", cfg["specialist_id"])
}

func TestBuildConfig_RateLimit(t *testing.T) {
	form := url.Values{
		"config_max_messages":  {"50"},
		"config_period_hours":  {"24"},
	}
	cfg := buildConfig("rate_limit", form)
	assert.Equal(t, float64(50), cfg["max_messages"])
	assert.Equal(t, float64(24), cfg["period_hours"])
}

func TestBuildConfig_DetectProduct(t *testing.T) {
	form := url.Values{
		"config_switch_specialist": {"true"},
	}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, true, cfg["switch_specialist"])
}

func TestBuildConfig_DetectProduct_Unchecked(t *testing.T) {
	form := url.Values{}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, false, cfg["switch_specialist"])
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/automation/interfaces/http/ -run TestBuildConfig -v`
Expected: FAIL — `buildConfig` not defined

- [ ] **Step 3: Create PageHandler with buildConfig and constructor**

```go
// internal/automation/interfaces/http/page_handler.go
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
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
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
	case "auto_message":
		tmpl := str("template")
		if len(tmpl) > 40 {
			tmpl = tmpl[:40] + "..."
		}
		return tmpl
	case "auto_note":
		tmpl := str("template")
		if len(tmpl) > 40 {
			tmpl = tmpl[:40] + "..."
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/automation/interfaces/http/ -run TestBuildConfig -v`
Expected: PASS (all 8 tests)

- [ ] **Step 5: Update ModuleDeps and module wiring**

Modify `internal/automation/module.go` — add new fields to `ModuleDeps` and create `PageHandler` in `NewModule`:

Add to `ModuleDeps` struct:
```go
ListFunnelsUC  *funnelapp.ListFunnelsUseCase
SpecialistRepo specialistdomain.SpecialistRepository
SpecTenantRepo specialistdomain.SpecialistTenantRepository
```

Add imports:
```go
specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
```

Add `pageHandler` field to `Module` struct:
```go
type Module struct {
	handler     *autohttp.Handler
	pageHandler *autohttp.PageHandler
	engine      *application.AutomationEngine
	ticker      *application.ExpirationTicker
}
```

In `NewModule`, after the existing `handler` creation, add:
```go
pageHandler := autohttp.NewPageHandler(crudUC, deps.ListFunnelsUC, deps.ColumnRepo, deps.SpecialistRepo, deps.SpecTenantRepo, log)
```

Update the return:
```go
return &Module{handler: handler, pageHandler: pageHandler, engine: engine, ticker: ticker}
```

Update `RegisterRoutes`:
```go
func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
	m.pageHandler.RegisterPageRoutes(router, mw.Auth, mw.Tenant, mw.RequirePermission)
}
```

- [ ] **Step 6: Add RegisterPageRoutes to routes.go**

Modify `internal/automation/interfaces/http/routes.go` — add the page routes method:

```go
// RegisterPageRoutes wires the automation HTML page routes.
func (h *PageHandler) RegisterPageRoutes(
	router *gin.Engine,
	authMw gin.HandlerFunc,
	tenantMw gin.HandlerFunc,
	requirePerm func(string, string) gin.HandlerFunc,
) {
	pages := router.Group("/tenant/automations")
	pages.Use(authMw, tenantMw, requirePerm("automations", "manage"))
	pages.GET("", h.ListPage)
	pages.GET("/table", h.RenderTable)
	pages.GET("/fields", h.RenderFields)
	pages.GET("/:id/form", h.RenderEditForm)
	pages.POST("", h.HandleCreate)
	pages.PUT("/:id", h.HandleUpdate)
	pages.DELETE("/:id", h.HandleDelete)
	pages.POST("/:id/toggle", h.HandleToggle)
	pages.GET("/:id/logs", h.RenderLogs)
}
```

- [ ] **Step 7: Add stub handler methods so the project compiles**

Add to `internal/automation/interfaces/http/page_handler.go`:

```go
func (h *PageHandler) ListPage(c *gin.Context)       { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderTable(c *gin.Context)     { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderFields(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderEditForm(c *gin.Context)  { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleCreate(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleUpdate(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleDelete(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) HandleToggle(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *PageHandler) RenderLogs(c *gin.Context)      { c.Status(http.StatusNotImplemented) }
```

- [ ] **Step 8: Update main.go to pass new deps**

Modify `cmd/api/main.go` — where automation module is created (around line 140), add the new deps:

```go
automationMod := automation.NewModule(db, automation.ModuleDeps{
	MoveLeadUC:     funnelMod.MoveLeadUC(),
	LeadRepo:       funnelMod.LeadRepo(),
	ColumnRepo:     funnelMod.ColumnRepo(),
	NoteRepo:       funnelMod.NoteRepo(),
	NotifyService:  notificationMod.NotifyService(),
	DB:             db,
	ListFunnelsUC:  funnelMod.ListFunnelsUC(),
	SpecialistRepo: specialistMod.SpecialistRepo(),
	SpecTenantRepo: specialistMod.SpecTenantRepo(),
}, log)
```

- [ ] **Step 9: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles successfully

- [ ] **Step 10: Run all existing tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/automation/... -v`
Expected: all existing tests pass + new buildConfig tests pass

- [ ] **Step 11: Commit**

```bash
git add internal/automation/interfaces/http/page_handler.go \
       internal/automation/interfaces/http/page_handler_test.go \
       internal/automation/interfaces/http/routes.go \
       internal/automation/module.go \
       cmd/api/main.go
git commit -m "feat(F09): scaffold PageHandler with buildConfig, routes and module wiring"
```

---

### Task 2: Sidebar + list page + table templates

**Files:**
- Modify: `web/templates/partials/tenant_sidebar.html`
- Create: `web/templates/automation/list.html`
- Create: `web/templates/automation/table.html`
- Modify: `internal/automation/interfaces/http/page_handler.go` (implement ListPage, RenderTable)

- [ ] **Step 1: Add "Automações" to tenant sidebar**

Modify `web/templates/partials/tenant_sidebar.html` — add after the Produtos `<a>` tag and before `</nav>`:

```html
        <a href="/tenant/automations" class="{{if .ActiveNav}}{{if eq .ActiveNav "automations"}}active{{end}}{{end}}">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>
            Automações
        </a>
```

- [ ] **Step 2: Create list.html template**

Create `web/templates/automation/list.html`:

```html
{{define "automation/list.html"}}
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Automações — CRM Juridico</title>
    <link rel="stylesheet" href="/static/css/main.css">
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" crossorigin="anonymous"></script>
</head>
<body>
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" "automations")}}
    <main class="admin-content">
        <div class="page-header">
            <div>
                <h1>Automações</h1>
            </div>
        </div>
        <div class="admin-table-wrapper">
            <div class="filters-bar">
                <select name="funnel_id" id="funnel-select"
                        hx-get="/tenant/automations/table"
                        hx-trigger="change"
                        hx-target="#automations-table"
                        hx-swap="innerHTML">
                    {{range .Funnels}}
                    <option value="{{.ID}}" {{if eq .ID $.CurrentFunnelID}}selected{{end}}>{{.Name}}</option>
                    {{end}}
                </select>
                <button class="btn btn-primary" onclick="document.getElementById('modal-title').textContent='Nova Automação';openModal('automation-modal')"
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" width="18" height="18"><path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4"/></svg>
                    Nova Automação
                </button>
            </div>
            <div id="automations-table"
                 hx-trigger="refreshTable from:body"
                 hx-get="/tenant/automations/table?funnel_id={{.CurrentFunnelID}}"
                 hx-swap="innerHTML">
                {{template "automation/table.html" .}}
            </div>
        </div>
    </main>
</div>

<!-- Create/Edit Modal -->
<div id="automation-modal" class="modal-overlay" style="display:none"
     onclick="if(event.target===this)closeModal('automation-modal')">
    <div class="modal-card modal-card-lg">
        <h2 id="modal-title">Nova Automação</h2>
        <div id="modal-body">
            {{template "automation/modal_form.html" .}}
        </div>
    </div>
</div>

<!-- Logs Modal -->
<div id="logs-modal" class="modal-overlay" style="display:none"
     onclick="if(event.target===this)closeModal('logs-modal')">
    <div class="modal-card modal-card-lg">
        <h2>Logs de Execução</h2>
        <div id="logs-body">
            <p class="text-muted">Carregando...</p>
        </div>
    </div>
</div>

<script src="/static/js/admin.js"></script>
<script>
document.body.addEventListener("refreshTable", function() {
    closeModal('automation-modal');
    // Update the hx-get URL on the table to use the currently selected funnel
    var funnelId = document.getElementById('funnel-select').value;
    var table = document.getElementById('automations-table');
    table.setAttribute('hx-get', '/tenant/automations/table?funnel_id=' + funnelId);
    htmx.process(table);
});
document.body.addEventListener("showLogs", function() {
    openModal('logs-modal');
});
</script>
</body>
</html>
{{end}}
```

- [ ] **Step 3: Create table.html template**

Create `web/templates/automation/table.html`:

```html
{{define "automation/table.html"}}
{{if .Automations}}
<table class="data-table">
    <thead>
        <tr>
            <th>Tipo</th>
            <th>Coluna</th>
            <th>Configuração</th>
            <th>Prioridade</th>
            <th>Status</th>
            <th></th>
        </tr>
    </thead>
    <tbody>
        {{range .Automations}}
        <tr>
            <td><strong>{{.TypeLabel}}</strong></td>
            <td>{{if .ColumnName}}{{.ColumnName}}{{else}}—{{end}}</td>
            <td class="text-truncate" style="max-width:250px">{{.ConfigSummary}}</td>
            <td>{{.Priority}}</td>
            <td>
                {{if .Active}}
                <span class="badge badge-active" style="cursor:pointer"
                      hx-post="/tenant/automations/{{.ID}}/toggle"
                      hx-target="#automations-table"
                      hx-swap="innerHTML"
                      hx-include="#funnel-select"
                      title="Clique para desativar">● Ativo</span>
                {{else}}
                <span class="badge badge-inactive" style="cursor:pointer"
                      hx-post="/tenant/automations/{{.ID}}/toggle"
                      hx-target="#automations-table"
                      hx-swap="innerHTML"
                      hx-include="#funnel-select"
                      title="Clique para ativar">○ Inativo</span>
                {{end}}
            </td>
            <td>
                <button class="btn btn-sm btn-outline"
                        hx-get="/tenant/automations/{{.ID}}/form"
                        hx-target="#modal-body"
                        hx-swap="innerHTML"
                        onclick="document.getElementById('modal-title').textContent='Editar Automação';openModal('automation-modal')"
                        title="Editar">✏️</button>
                <button class="btn btn-sm btn-outline"
                        hx-get="/tenant/automations/{{.ID}}/logs?limit=20&offset=0"
                        hx-target="#logs-body"
                        hx-swap="innerHTML"
                        onclick="openModal('logs-modal')"
                        title="Logs">📊</button>
                <button class="btn btn-sm btn-outline"
                        hx-delete="/tenant/automations/{{.ID}}"
                        hx-target="#automations-table"
                        hx-swap="innerHTML"
                        hx-include="#funnel-select"
                        hx-confirm="Excluir esta automação?"
                        title="Excluir"
                        style="color:#e74c3c">🗑️</button>
            </td>
        </tr>
        {{end}}
    </tbody>
</table>
{{else}}
<div class="empty-state" style="padding:3rem 1rem">
    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" width="48" height="48" style="color:#cbd5e0;margin:0 auto 1rem;display:block">
        <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z"/>
    </svg>
    Nenhuma automação configurada para este funil
</div>
{{end}}
{{end}}
```

- [ ] **Step 4: Implement ListPage and RenderTable handlers**

Replace the `ListPage` and `RenderTable` stubs in `page_handler.go`:

```go
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
```

- [ ] **Step 5: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles successfully

- [ ] **Step 6: Commit**

```bash
git add web/templates/partials/tenant_sidebar.html \
       web/templates/automation/list.html \
       web/templates/automation/table.html \
       internal/automation/interfaces/http/page_handler.go
git commit -m "feat(F09): add sidebar item, list page and table templates for automations"
```

---

### Task 3: Toggle + delete handlers

**Files:**
- Modify: `internal/automation/interfaces/http/page_handler.go`

- [ ] **Step 1: Implement HandleToggle**

Replace the `HandleToggle` stub in `page_handler.go`:

```go
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
```

- [ ] **Step 2: Implement HandleDelete**

Replace the `HandleDelete` stub in `page_handler.go`:

```go
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
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles

- [ ] **Step 4: Commit**

```bash
git add internal/automation/interfaces/http/page_handler.go
git commit -m "feat(F09): implement toggle and delete handlers for automations"
```

---

### Task 4: Modal form template + dynamic fields

**Files:**
- Create: `web/templates/automation/modal_form.html`
- Create: `web/templates/automation/fields_expiration.html`
- Create: `web/templates/automation/fields_move_funnel.html`
- Create: `web/templates/automation/fields_auto_message.html`
- Create: `web/templates/automation/fields_auto_note.html`
- Create: `web/templates/automation/fields_switch_specialist.html`
- Create: `web/templates/automation/fields_rate_limit.html`
- Create: `web/templates/automation/fields_detect_product.html`
- Modify: `internal/automation/interfaces/http/page_handler.go` (RenderFields, RenderEditForm)

- [ ] **Step 1: Create modal_form.html**

```html
{{define "automation/modal_form.html"}}
{{if .FormError}}
<div class="alert alert-danger" style="margin-bottom:12px">{{.FormError}}</div>
{{end}}
<form {{if .IsEdit}}
          hx-put="/tenant/automations/{{.Automation.ID}}"
      {{else}}
          hx-post="/tenant/automations?funnel_id={{.CurrentFunnelID}}"
      {{end}}
      hx-target="#modal-body"
      hx-swap="innerHTML"
      hx-disabled-elt="button[type='submit']">

    <input type="hidden" name="funnel_id" value="{{.CurrentFunnelID}}">

    <div class="form-group">
        <label for="column_id">Coluna</label>
        <select id="column_id" name="column_id">
            <option value="">— Nenhuma (global) —</option>
            {{range .Columns}}
            <option value="{{.ID}}" {{if eq .ID $.SelectedColumnID}}selected{{end}}>{{.Name}}</option>
            {{end}}
        </select>
    </div>

    <div class="form-group">
        <label for="type">Tipo</label>
        <select id="type" name="type"
                hx-get="/tenant/automations/fields"
                hx-trigger="change"
                hx-target="#dynamic-fields"
                hx-swap="innerHTML"
                hx-include="[name='funnel_id']"
                {{if .IsEdit}}disabled{{end}}>
            <option value="expiration" {{if eq .SelectedType "expiration"}}selected{{end}}>⏰ Exclusão por tempo</option>
            <option value="move_funnel" {{if eq .SelectedType "move_funnel"}}selected{{end}}>🔀 Mover para funil</option>
            <option value="auto_message" {{if eq .SelectedType "auto_message"}}selected{{end}}>💬 Mensagem automática</option>
            <option value="auto_note" {{if eq .SelectedType "auto_note"}}selected{{end}}>📋 Anotação automática</option>
            <option value="switch_specialist" {{if eq .SelectedType "switch_specialist"}}selected{{end}}>👤 Trocar especialista</option>
            <option value="rate_limit" {{if eq .SelectedType "rate_limit"}}selected{{end}}>🚫 Limite de mensagens</option>
            <option value="detect_product" {{if eq .SelectedType "detect_product"}}selected{{end}}>🔍 Detectar produto</option>
        </select>
        {{if .IsEdit}}<input type="hidden" name="type" value="{{.SelectedType}}">{{end}}
    </div>

    <div id="dynamic-fields"
         hx-get="/tenant/automations/fields?type={{if .SelectedType}}{{.SelectedType}}{{else}}expiration{{end}}&funnel_id={{.CurrentFunnelID}}{{if .IsEdit}}&automation_id={{.Automation.ID}}{{end}}"
         hx-trigger="load"
         hx-swap="innerHTML">
        <p class="text-muted">Carregando campos...</p>
    </div>

    <div class="form-group">
        <label for="priority">Prioridade</label>
        <input type="number" id="priority" name="priority" value="{{.SelectedPriority}}" min="0">
        <small class="text-muted">Menor número = executa primeiro</small>
    </div>

    <div class="modal-actions">
        <button type="button" class="btn btn-secondary" onclick="closeModal('automation-modal')">Cancelar</button>
        <button type="submit" class="btn btn-primary">
            <span class="btn-text">{{if .IsEdit}}Salvar{{else}}Criar{{end}}</span>
            <span class="btn-loading-text">Salvando...</span>
            <span class="spinner"></span>
        </button>
    </div>
</form>
{{end}}
```

- [ ] **Step 2: Create field templates (all 7)**

Create `web/templates/automation/fields_expiration.html`:
```html
{{define "automation/fields_expiration.html"}}
<div class="form-group">
    <label for="config_action">Ação</label>
    <select id="config_action" name="config_action" required>
        <option value="archive" {{if eq .ConfigAction "archive"}}selected{{end}}>Arquivar (mover para coluna perdidos)</option>
        <option value="delete" {{if eq .ConfigAction "delete"}}selected{{end}}>Excluir definitivamente</option>
    </select>
</div>
<div class="form-group">
    <label for="config_duration_hours">Tempo (horas)</label>
    <input type="number" id="config_duration_hours" name="config_duration_hours"
           value="{{if .ConfigDurationHours}}{{.ConfigDurationHours}}{{else}}48{{end}}" min="1" step="1" required>
</div>
{{end}}
```

Create `web/templates/automation/fields_move_funnel.html`:
```html
{{define "automation/fields_move_funnel.html"}}
<div class="form-group">
    <label for="config_target_funnel_id">Funil destino</label>
    <select id="config_target_funnel_id" name="config_target_funnel_id" required>
        <option value="">Selecione...</option>
        {{range .Funnels}}
        <option value="{{.ID}}" {{if eq .ID $.ConfigTargetFunnelID}}selected{{end}}>{{.Name}}</option>
        {{end}}
    </select>
</div>
<div class="form-group">
    <label for="config_target_column_id">Coluna destino</label>
    <input type="text" id="config_target_column_id" name="config_target_column_id"
           value="{{.ConfigTargetColumnID}}" placeholder="ID da coluna (deixe vazio para coluna de entrada)">
    <small class="text-muted">Deixe vazio para usar a coluna de entrada do funil destino</small>
</div>
{{end}}
```

Create `web/templates/automation/fields_auto_message.html`:
```html
{{define "automation/fields_auto_message.html"}}
<div class="form-group">
    <label for="config_template">Template da mensagem</label>
    <textarea id="config_template" name="config_template" rows="4"
              style="font-family:monospace" required
              placeholder="Olá {{"{{"}}nome{{"}}"}}, recebemos sua mensagem...">{{.ConfigTemplate}}</textarea>
    <small class="text-muted">Variáveis disponíveis: {{"{{"}}nome{{"}}"}}, {{"{{"}}produto{{"}}"}}, {{"{{"}}especialista{{"}}"}}</small>
</div>
{{end}}
```

Create `web/templates/automation/fields_auto_note.html`:
```html
{{define "automation/fields_auto_note.html"}}
<div class="form-group">
    <label for="config_template">Template da anotação</label>
    <textarea id="config_template" name="config_template" rows="3" required
              placeholder="Lead movido automaticamente para esta coluna">{{.ConfigTemplate}}</textarea>
</div>
{{end}}
```

Create `web/templates/automation/fields_switch_specialist.html`:
```html
{{define "automation/fields_switch_specialist.html"}}
<div class="form-group">
    <label for="config_specialist_id">Especialista</label>
    <select id="config_specialist_id" name="config_specialist_id" required>
        <option value="">Selecione...</option>
        {{range .Specialists}}
        <option value="{{.ID}}" {{if eq .ID $.ConfigSpecialistID}}selected{{end}}>{{.Name}}</option>
        {{end}}
    </select>
</div>
{{end}}
```

Create `web/templates/automation/fields_rate_limit.html`:
```html
{{define "automation/fields_rate_limit.html"}}
<div class="form-group">
    <label for="config_max_messages">Limite de mensagens</label>
    <input type="number" id="config_max_messages" name="config_max_messages"
           value="{{if .ConfigMaxMessages}}{{.ConfigMaxMessages}}{{else}}50{{end}}" min="1" required>
</div>
<div class="form-group">
    <label for="config_period_hours">Período (horas)</label>
    <input type="number" id="config_period_hours" name="config_period_hours"
           value="{{if .ConfigPeriodHours}}{{.ConfigPeriodHours}}{{else}}24{{end}}" min="1" required>
    <small class="text-muted">Ex: 50 mensagens a cada 24 horas</small>
</div>
{{end}}
```

Create `web/templates/automation/fields_detect_product.html`:
```html
{{define "automation/fields_detect_product.html"}}
<div class="form-group">
    <label>
        <input type="checkbox" name="config_switch_specialist" value="true"
               {{if .ConfigSwitchSpecialist}}checked{{end}}>
        Trocar especialista automaticamente ao detectar produto
    </label>
    <small class="text-muted">Se ativado, o lead será redirecionado para o especialista do produto detectado</small>
</div>
{{end}}
```

- [ ] **Step 3: Implement RenderFields handler**

Replace the `RenderFields` stub in `page_handler.go`:

```go
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
```

- [ ] **Step 4: Implement RenderEditForm handler**

Replace the `RenderEditForm` stub in `page_handler.go`:

```go
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

	// Add type-specific config values
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
```

- [ ] **Step 5: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles

- [ ] **Step 6: Commit**

```bash
git add web/templates/automation/modal_form.html \
       web/templates/automation/fields_*.html \
       internal/automation/interfaces/http/page_handler.go
git commit -m "feat(F09): add modal form and dynamic field templates for all 7 automation types"
```

---

### Task 5: Create + update form handlers

**Files:**
- Modify: `internal/automation/interfaces/http/page_handler.go`

- [ ] **Step 1: Write test for form parsing edge cases**

Add to `page_handler_test.go`:

```go
func TestBuildConfig_Expiration_InvalidNumber(t *testing.T) {
	form := url.Values{
		"config_action":         {"archive"},
		"config_duration_hours": {"not-a-number"},
	}
	cfg := buildConfig("expiration", form)
	assert.Equal(t, "archive", cfg["action"])
	_, hasDuration := cfg["duration_hours"]
	assert.False(t, hasDuration, "should not set duration_hours for invalid number")
}

func TestBuildConfig_UnknownType(t *testing.T) {
	form := url.Values{"config_foo": {"bar"}}
	cfg := buildConfig("unknown_type", form)
	assert.Empty(t, cfg)
}
```

- [ ] **Step 2: Run tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/automation/interfaces/http/ -run TestBuildConfig -v`
Expected: PASS

- [ ] **Step 3: Implement HandleCreate**

Replace the `HandleCreate` stub in `page_handler.go`:

```go
// HandleCreate handles POST /tenant/automations — creates automation and triggers table refresh.
func (h *PageHandler) HandleCreate(c *gin.Context) {
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
```

- [ ] **Step 4: Implement HandleUpdate**

Replace the `HandleUpdate` stub in `page_handler.go`:

```go
// HandleUpdate handles PUT /tenant/automations/:id — updates automation and triggers table refresh.
func (h *PageHandler) HandleUpdate(c *gin.Context) {
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
```

- [ ] **Step 5: Verify compilation and tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./... && go test ./internal/automation/interfaces/http/ -v`
Expected: compiles and all tests pass

- [ ] **Step 6: Commit**

```bash
git add internal/automation/interfaces/http/page_handler.go \
       internal/automation/interfaces/http/page_handler_test.go
git commit -m "feat(F09): implement create and update form handlers with config parsing"
```

---

### Task 6: Logs modal

**Files:**
- Create: `web/templates/automation/modal_logs.html`
- Modify: `internal/automation/interfaces/http/page_handler.go` (RenderLogs)

- [ ] **Step 1: Create modal_logs.html**

```html
{{define "automation/modal_logs.html"}}
{{if .Logs}}
<table class="data-table">
    <thead>
        <tr>
            <th>Data</th>
            <th>Lead</th>
            <th>Status</th>
            <th>Erro</th>
        </tr>
    </thead>
    <tbody>
        {{range .Logs}}
        <tr>
            <td>{{.ExecutedAt.Format "02/01/2006 15:04"}}</td>
            <td style="font-family:monospace;font-size:12px">{{.LeadID}}</td>
            <td>
                {{if eq .Status "success"}}
                <span class="badge badge-active">success</span>
                {{else}}
                <span class="badge badge-inactive">{{.Status}}</span>
                {{end}}
            </td>
            <td>{{if .ErrorMessage}}<span style="color:#e74c3c;font-size:12px">{{.ErrorMessage}}</span>{{else}}<span class="text-muted">—</span>{{end}}</td>
        </tr>
        {{end}}
    </tbody>
</table>
<div class="pagination">
    <span class="pagination-info">Mostrando {{.StartItem}}-{{.EndItem}}</span>
    <div class="pagination-buttons">
        {{if .HasPrev}}
        <button class="btn btn-sm btn-outline"
                hx-get="/tenant/automations/{{.AutomationID}}/logs?limit={{.Limit}}&offset={{.PrevOffset}}"
                hx-target="#logs-body"
                hx-swap="innerHTML">
            Anterior
        </button>
        {{end}}
        {{if .HasNext}}
        <button class="btn btn-sm btn-outline"
                hx-get="/tenant/automations/{{.AutomationID}}/logs?limit={{.Limit}}&offset={{.NextOffset}}"
                hx-target="#logs-body"
                hx-swap="innerHTML">
            Próximo
        </button>
        {{end}}
    </div>
</div>
{{else}}
<div class="empty-state" style="padding:2rem 1rem">
    <p class="text-muted">Nenhum log de execução encontrado</p>
</div>
{{end}}
{{end}}
```

- [ ] **Step 2: Implement RenderLogs handler**

Replace the `RenderLogs` stub in `page_handler.go`:

```go
// RenderLogs handles GET /tenant/automations/:id/logs — returns logs table fragment.
func (h *PageHandler) RenderLogs(c *gin.Context) {
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

	c.HTML(http.StatusOK, "automation/modal_logs.html", gin.H{
		"Logs":         logs,
		"AutomationID": id,
		"Limit":        limit,
		"StartItem":    offset + 1,
		"EndItem":      offset + len(logs),
		"HasPrev":      offset > 0,
		"HasNext":      hasNext,
		"PrevOffset":   offset - limit,
		"NextOffset":   offset + limit,
	})
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles

- [ ] **Step 4: Commit**

```bash
git add web/templates/automation/modal_logs.html \
       internal/automation/interfaces/http/page_handler.go
git commit -m "feat(F09): add execution logs modal with pagination"
```

---

### Task 7: Remove stubs + final verification

**Files:**
- Modify: `internal/automation/interfaces/http/page_handler.go` (clean up)

- [ ] **Step 1: Remove all stub methods**

Verify that all 9 stub methods (`ListPage`, `RenderTable`, `RenderFields`, `RenderEditForm`, `HandleCreate`, `HandleUpdate`, `HandleDelete`, `HandleToggle`, `RenderLogs`) have been replaced with real implementations. Remove any remaining `http.StatusNotImplemented` stubs.

- [ ] **Step 2: Run all tests**

Run: `cd /home/sasrgita/projects/crm_juridico && go test ./internal/automation/... -v`
Expected: all tests pass

- [ ] **Step 3: Build and verify**

Run: `cd /home/sasrgita/projects/crm_juridico && go build ./...`
Expected: compiles with no errors

- [ ] **Step 4: Start the app and test manually**

Run: `cd /home/sasrgita/projects/crm_juridico && go run cmd/api/main.go`

Manual checks:
1. Navigate to `/tenant/automations` — page loads with funnel dropdown
2. Change funnel in dropdown — table refreshes via HTMX
3. Click "+ Nova Automação" — modal opens with form
4. Change type in form — dynamic fields update via HTMX
5. Create an automation — modal closes, table refreshes
6. Click status badge — toggles active/inactive
7. Click ✏️ — edit modal opens with pre-filled data
8. Click 📊 — logs modal opens
9. Click 🗑️ — confirm dialog, automation deleted

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat(F09): complete automation HTMX screens — list, CRUD, toggle, logs"
```

---

### Task 8: Update .http test file + feature doc

**Files:**
- Modify: `docs/features/F09-automacoes.md`
- Modify: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md`

- [ ] **Step 1: Mark Step 7 as done in F09**

In `docs/features/F09-automacoes.md`, change Step 7 items from `- [ ]` to `- [x]`.

- [ ] **Step 2: Update status.md**

Update `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md` to reflect that Step 7 (telas HTMX) is complete.

- [ ] **Step 3: Commit**

```bash
git add docs/features/F09-automacoes.md \
       docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md
git commit -m "docs(F09): mark Step 7 (automation screens) as complete"
```
