package http

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	aiapp "github.com/sasrgita/crm-juridico/internal/ai/application"
	aidomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

// ToolHandler handles admin UI routes for associating tools to a specialist.
type ToolHandler struct {
	toolRepo     domain.SpecialistToolRepository
	toolRegistry *aiapp.ToolRegistry
}

// NewToolHandler creates a ToolHandler.
func NewToolHandler(toolRepo domain.SpecialistToolRepository, toolRegistry *aiapp.ToolRegistry) *ToolHandler {
	return &ToolHandler{toolRepo: toolRepo, toolRegistry: toolRegistry}
}

// RegisterRoutes wires the tool management routes onto the /admin/specialists group.
func (h *ToolHandler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("/:id/tools", h.HandleList)
	admin.POST("/:id/tools", h.HandleAssociate)
	admin.DELETE("/:id/tools/:name", h.HandleDissociate)
}

// HandleList renders the tools section partial for a specialist.
func (h *ToolHandler) HandleList(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	associated, err := h.toolRepo.FindToolNamesBySpecialistID(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar tools"})
		return
	}

	associatedMap := make(map[string]bool, len(associated))
	for _, name := range associated {
		associatedMap[name] = true
	}

	allDefs := h.toolRegistry.Definitions()
	byCategory := groupByCategory(allDefs)

	c.HTML(http.StatusOK, "specialist/tools_section.html", gin.H{
		"SpecialistID":    id,
		"AllTools":        allDefs,
		"Associated":      associatedMap,
		"ToolsByCategory": byCategory,
	})
}

// HandleAssociate associates a tool with the specialist and re-renders the section.
func (h *ToolHandler) HandleAssociate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	toolName := c.PostForm("tool_name")
	if toolName == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "tool_name é obrigatório"})
		return
	}

	if err := h.toolRepo.Associate(c.Request.Context(), id, toolName); err != nil {
		if isTool, msg := mapToolError(err); isTool {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao associar tool"})
		return
	}

	h.HandleList(c)
}

// HandleDissociate dissociates a tool from the specialist and re-renders the section.
func (h *ToolHandler) HandleDissociate(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	toolName := c.Param("name")
	if err := h.toolRepo.Dissociate(c.Request.Context(), id, toolName); err != nil {
		if isTool, msg := mapToolError(err); isTool {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao remover tool"})
		return
	}

	h.HandleList(c)
}

// categoryEntry holds tools belonging to a single category for template rendering.
type categoryEntry struct {
	Label string
	Tools []aidomain.ToolDefinition
}

// groupByCategory sorts tools into ordered category buckets for the template.
func groupByCategory(defs []aidomain.ToolDefinition) []categoryEntry {
	order := []aidomain.ToolCategory{
		aidomain.ToolCategoryDataQuery,
		aidomain.ToolCategoryCRMAction,
		aidomain.ToolCategoryAutomation,
	}
	labels := map[aidomain.ToolCategory]string{
		aidomain.ToolCategoryDataQuery:  "Consulta de Dados",
		aidomain.ToolCategoryCRMAction:  "Ações no CRM",
		aidomain.ToolCategoryAutomation: "Automações",
	}

	buckets := make(map[aidomain.ToolCategory][]aidomain.ToolDefinition)
	for _, d := range defs {
		buckets[d.Category] = append(buckets[d.Category], d)
	}

	var result []categoryEntry
	for _, cat := range order {
		tools := buckets[cat]
		if len(tools) == 0 {
			continue
		}
		sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
		result = append(result, categoryEntry{
			Label: labels[cat],
			Tools: tools,
		})
	}
	return result
}

// mapToolError maps domain tool errors to user-facing messages.
// Returns (true, message) if the error was recognised, otherwise (false, "").
func mapToolError(err error) (bool, string) {
	switch {
	case err == nil:
		return false, ""
	case err == domain.ErrToolAlreadyAssociated:
		return true, "Tool já está associada a este especialista"
	case err == domain.ErrToolNotAssociated:
		return true, "Tool não está associada a este especialista"
	default:
		return false, ""
	}
}
