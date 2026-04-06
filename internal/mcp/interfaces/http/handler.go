package http

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/mcp/application"
	"github.com/sasrgita/crm-juridico/internal/mcp/domain"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

type Handler struct {
	createUC        *application.CreateMcpUseCase
	listUC          *application.ListMcpsUseCase
	getUC           *application.GetMcpUseCase
	updateUC        *application.UpdateMcpUseCase
	deleteUC        *application.DeleteMcpUseCase
	associateUC     *application.AssociateMcpUseCase
	dissociateUC    *application.DissociateMcpUseCase
	listSpecMcpsUC  *application.ListSpecialistMcpsUseCase
	listAvailableUC *application.ListAvailableMcpsUseCase
}

func NewHandler(
	createUC *application.CreateMcpUseCase,
	listUC *application.ListMcpsUseCase,
	getUC *application.GetMcpUseCase,
	updateUC *application.UpdateMcpUseCase,
	deleteUC *application.DeleteMcpUseCase,
	associateUC *application.AssociateMcpUseCase,
	dissociateUC *application.DissociateMcpUseCase,
	listSpecMcpsUC *application.ListSpecialistMcpsUseCase,
	listAvailableUC *application.ListAvailableMcpsUseCase,
) *Handler {
	return &Handler{
		createUC: createUC, listUC: listUC, getUC: getUC,
		updateUC: updateUC, deleteUC: deleteUC,
		associateUC: associateUC, dissociateUC: dissociateUC,
		listSpecMcpsUC: listSpecMcpsUC, listAvailableUC: listAvailableUC,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/mcps")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderList)
	admin.GET("/new", h.RenderCreateForm)
	admin.POST("", h.HandleCreate)
	admin.GET("/:id", h.RenderDetail)
	admin.GET("/:id/edit", h.RenderEditForm)
	admin.PUT("/:id", h.HandleUpdate)
	admin.DELETE("/:id", h.HandleDelete)

	specAdmin := router.Group("/admin/specialists")
	specAdmin.Use(authMw, adminMw)

	specAdmin.GET("/:id/mcps", h.HandleListSpecialistMcps)
	specAdmin.GET("/:id/mcps/available", h.HandleListAvailableMcps)
	specAdmin.POST("/:id/mcps", h.HandleAssociateMcps)
	specAdmin.DELETE("/:id/mcps/:mcp_id", h.HandleDissociateMcp)
}

func (h *Handler) RenderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	output, err := h.listUC.Execute(c.Request.Context(), application.ListMcpsInput{
		Search: c.Query("search"), Page: page, Limit: limit,
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "mcp/list.html", gin.H{"Error": "Erro ao carregar MCPs"})
		return
	}

	data := gin.H{
		"McpServers": output.McpServers,
		"Total":      output.Total,
		"Page":       output.Page,
		"Limit":      output.Limit,
		"Search":     c.Query("search"),
		"HasPrev":    output.Page > 1,
		"HasNext":    int64(output.Page*output.Limit) < output.Total,
		"PrevPage":   output.Page - 1,
		"NextPage":   output.Page + 1,
		"StartItem":  (output.Page-1)*output.Limit + 1,
		"EndItem":    min(int64(output.Page*output.Limit), output.Total),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "mcp/table.html", data)
		return
	}
	c.HTML(http.StatusOK, "mcp/list.html", data)
}

func (h *Handler) RenderCreateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "mcp/form.html", gin.H{"IsEdit": false, "Mcp": nil, "Error": ""})
}

func (h *Handler) HandleCreate(c *gin.Context) {
	input := application.CreateMcpInput{
		Name: c.PostForm("name"), URL: c.PostForm("url"),
		Config: c.PostForm("config"), Headers: c.PostForm("headers"),
	}

	output, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		c.HTML(http.StatusOK, "mcp/form.html", gin.H{
			"IsEdit": false, "Mcp": input, "Error": mapDomainError(err),
		})
		return
	}

	c.Header("HX-Redirect", "/admin/mcps/"+output.ID)
	c.Status(http.StatusOK)
}

func (h *Handler) RenderDetail(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.HTML(http.StatusBadRequest, "mcp/detail.html", gin.H{"Error": "ID inválido"})
		return
	}

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrMcpNotFound) {
			c.HTML(http.StatusNotFound, "mcp/detail.html", gin.H{"Error": "MCP não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "mcp/detail.html", gin.H{"Error": "Erro ao carregar MCP"})
		return
	}

	c.HTML(http.StatusOK, "mcp/detail.html", gin.H{"Mcp": output, "Error": ""})
}

func (h *Handler) RenderEditForm(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.HTML(http.StatusBadRequest, "mcp/form.html", gin.H{"Error": "ID inválido"})
		return
	}

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrMcpNotFound) {
			c.HTML(http.StatusNotFound, "mcp/form.html", gin.H{"Error": "MCP não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "mcp/form.html", gin.H{"Error": "Erro ao carregar MCP"})
		return
	}

	c.HTML(http.StatusOK, "mcp/form.html", gin.H{"IsEdit": true, "Mcp": output, "Error": ""})
}

func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")
	input := application.UpdateMcpInput{
		ID: id, Name: c.PostForm("name"), URL: c.PostForm("url"),
		Config: c.PostForm("config"), Headers: c.PostForm("headers"),
	}

	err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		c.HTML(http.StatusOK, "mcp/form.html", gin.H{
			"IsEdit": true, "Mcp": input, "Error": mapDomainError(err),
		})
		return
	}

	c.Header("HX-Redirect", "/admin/mcps/"+id)
	c.Status(http.StatusOK)
}

func (h *Handler) HandleDelete(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	if err := h.deleteUC.Execute(c.Request.Context(), id); err != nil {
		if errors.Is(err, domain.ErrMcpNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "MCP não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao excluir MCP"})
		return
	}

	c.Header("HX-Redirect", "/admin/mcps")
	c.Status(http.StatusOK)
}

// --- Specialist-MCP association handlers ---

func (h *Handler) HandleListSpecialistMcps(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	items, err := h.listSpecMcpsUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar MCPs"})
		return
	}
	c.HTML(http.StatusOK, "specialist/mcps_section.html", gin.H{"Mcps": items, "SpecialistID": id})
}

func (h *Handler) HandleListAvailableMcps(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	items, err := h.listAvailableUC.Execute(c.Request.Context(), id, c.Query("search"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar MCPs disponíveis"})
		return
	}
	c.HTML(http.StatusOK, "specialist/mcps_modal.html", gin.H{"AvailableMcps": items, "SpecialistID": id, "Search": c.Query("search")})
}

func (h *Handler) HandleAssociateMcps(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	mcpIDs := c.PostFormArray("mcp_ids[]")
	if len(mcpIDs) == 0 {
		mcpIDs = c.PostFormArray("mcp_ids")
	}
	if len(mcpIDs) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Selecione pelo menos um MCP"})
		return
	}

	err := h.associateUC.Execute(c.Request.Context(), application.AssociateMcpInput{SpecialistID: id, McpIDs: mcpIDs})
	if err != nil {
		if errors.Is(err, specialistdomain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, specialistdomain.ErrSpecialistInactive) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
			return
		}
		if errors.Is(err, application.ErrAssociationBatchTooLarge) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Máximo de %d MCPs por associação", application.MaxAssociationBatchSize)})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.HandleListSpecialistMcps(c)
}

func (h *Handler) HandleDissociateMcp(c *gin.Context) {
	id := c.Param("id")
	mcpID := c.Param("mcp_id")

	if err := h.dissociateUC.Execute(c.Request.Context(), id, mcpID); err != nil {
		if errors.Is(err, domain.ErrMcpNotAssociated) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "MCP não está associado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao desassociar MCP"})
		return
	}

	h.HandleListSpecialistMcps(c)
}

func mapDomainError(err error) string {
	switch {
	case errors.Is(err, domain.ErrMcpNameRequired):
		return "Nome é obrigatório"
	case errors.Is(err, domain.ErrMcpNameTooLong):
		return "Nome excede o tamanho máximo"
	case errors.Is(err, domain.ErrMcpURLRequired):
		return "URL é obrigatória"
	case errors.Is(err, domain.ErrMcpURLInvalid):
		return "URL inválida"
	case errors.Is(err, domain.ErrMcpConfigTooLong):
		return "Configuração excede o tamanho máximo"
	case errors.Is(err, domain.ErrMcpHeadersTooLong):
		return "Headers excede o tamanho máximo"
	case errors.Is(err, domain.ErrMcpNotFound):
		return "MCP não encontrado"
	default:
		return "Erro interno"
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
