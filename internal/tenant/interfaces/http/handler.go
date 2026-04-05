package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/tenant/application"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type Handler struct {
	createUC          *application.CreateTenantUseCase
	listUC            *application.ListTenantsUseCase
	getUC             *application.GetTenantUseCase
	updateUC          *application.UpdateTenantUseCase
	deactivateUC      *application.DeactivateTenantUseCase
	blockUC           *application.BlockTenantUseCase
	unblockUC         *application.UnblockTenantUseCase
	getBlockHistoryUC *application.GetBlockHistoryUseCase
}

func NewHandler(
	createUC *application.CreateTenantUseCase,
	listUC *application.ListTenantsUseCase,
	getUC *application.GetTenantUseCase,
	updateUC *application.UpdateTenantUseCase,
	deactivateUC *application.DeactivateTenantUseCase,
	blockUC *application.BlockTenantUseCase,
	unblockUC *application.UnblockTenantUseCase,
	getBlockHistoryUC *application.GetBlockHistoryUseCase,
) *Handler {
	return &Handler{
		createUC:          createUC,
		listUC:            listUC,
		getUC:             getUC,
		updateUC:          updateUC,
		deactivateUC:      deactivateUC,
		blockUC:           blockUC,
		unblockUC:         unblockUC,
		getBlockHistoryUC: getBlockHistoryUC,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/tenants")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderList)
	admin.GET("/new", h.RenderCreateForm)
	admin.POST("", h.HandleCreate)
	admin.GET("/:id", h.RenderDetail)
	admin.GET("/:id/edit", h.RenderEditForm)
	admin.PUT("/:id", h.HandleUpdate)
	admin.DELETE("/:id", h.HandleDeactivate)
	admin.POST("/:id/block", h.HandleBlock)
	admin.POST("/:id/unblock", h.HandleUnblock)
	admin.GET("/:id/block-history", h.HandleGetBlockHistory)
}

func (h *Handler) RenderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	input := application.ListTenantsInput{
		Search: c.Query("search"),
		Status: c.Query("status"),
		Type:   c.Query("type"),
		Page:   page,
		Limit:  limit,
	}

	output, err := h.listUC.Execute(c.Request.Context(), input)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "tenant/list.html", gin.H{"Error": "Erro ao carregar tenants"})
		return
	}

	data := gin.H{
		"Tenants":    output.Tenants,
		"Total":      output.Total,
		"Page":       output.Page,
		"Limit":      output.Limit,
		"Search":     input.Search,
		"Status":     input.Status,
		"Type":       input.Type,
		"HasPrev":    output.Page > 1,
		"HasNext":    int64(output.Page*output.Limit) < output.Total,
		"PrevPage":   output.Page - 1,
		"NextPage":   output.Page + 1,
		"StartItem":  (output.Page-1)*output.Limit + 1,
		"EndItem":    min(int64(output.Page*output.Limit), output.Total),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "tenant/table.html", data)
		return
	}
	c.HTML(http.StatusOK, "tenant/list.html", data)
}

func (h *Handler) RenderCreateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "tenant/form.html", gin.H{
		"IsEdit": false,
		"Tenant": nil,
		"Error":  "",
	})
}

func (h *Handler) HandleCreate(c *gin.Context) {
	input := application.CreateTenantInput{
		Name:     c.PostForm("name"),
		Type:     c.PostForm("type"),
		Document: c.PostForm("document"),
	}

	output, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.renderFormError(c, false, input.Name, input.Type, input.Document, "", mapDomainError(err))
		return
	}

	c.Header("HX-Redirect", "/admin/tenants/"+output.ID)
	c.Status(http.StatusOK)
}

func (h *Handler) RenderDetail(c *gin.Context) {
	id := c.Param("id")

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.HTML(http.StatusNotFound, "tenant/detail.html", gin.H{"Error": "Tenant não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "tenant/detail.html", gin.H{"Error": "Erro ao carregar tenant"})
		return
	}

	c.HTML(http.StatusOK, "tenant/detail.html", gin.H{
		"Tenant": output,
		"Error":  "",
	})
}

func (h *Handler) RenderEditForm(c *gin.Context) {
	id := c.Param("id")

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.HTML(http.StatusNotFound, "tenant/form.html", gin.H{"Error": "Tenant não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "tenant/form.html", gin.H{"Error": "Erro ao carregar tenant"})
		return
	}

	c.HTML(http.StatusOK, "tenant/form.html", gin.H{
		"IsEdit": true,
		"Tenant": output,
		"Error":  "",
	})
}

func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")

	input := application.UpdateTenantInput{
		ID:       id,
		Name:     c.PostForm("name"),
		Type:     c.PostForm("type"),
		Document: c.PostForm("document"),
	}

	_, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.renderFormError(c, true, input.Name, input.Type, input.Document, id, mapDomainError(err))
		return
	}

	c.Header("HX-Redirect", "/admin/tenants/"+id)
	c.Status(http.StatusOK)
}

func (h *Handler) HandleDeactivate(c *gin.Context) {
	id := c.Param("id")

	err := h.deactivateUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tenant não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	c.Header("HX-Redirect", "/admin/tenants")
	c.Status(http.StatusOK)
}

func (h *Handler) HandleBlock(c *gin.Context) {
	id := c.Param("id")
	reason := c.PostForm("reason")
	performedBy := c.GetString("user_id")

	err := h.blockUC.Execute(c.Request.Context(), application.BlockTenantInput{
		ID:          id,
		Reason:      reason,
		PerformedBy: performedBy,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tenant não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.renderDetailAfterAction(c, id, "Tenant bloqueado com sucesso")
}

func (h *Handler) HandleUnblock(c *gin.Context) {
	id := c.Param("id")
	reason := c.PostForm("reason")
	performedBy := c.GetString("user_id")

	err := h.unblockUC.Execute(c.Request.Context(), application.UnblockTenantInput{
		ID:          id,
		Reason:      reason,
		PerformedBy: performedBy,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tenant não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.renderDetailAfterAction(c, id, "Tenant desbloqueado com sucesso")
}

func (h *Handler) HandleGetBlockHistory(c *gin.Context) {
	id := c.Param("id")

	history, err := h.getBlockHistoryUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tenant não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar histórico"})
		return
	}

	c.HTML(http.StatusOK, "tenant/block_history.html", gin.H{
		"History":  history,
		"TenantID": id,
	})
}

func (h *Handler) renderDetailAfterAction(c *gin.Context, id, successMsg string) {
	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar tenant"})
		return
	}

	c.HTML(http.StatusOK, "tenant/detail.html", gin.H{
		"Tenant":  output,
		"Success": successMsg,
		"Error":   "",
	})
}

func (h *Handler) renderFormError(c *gin.Context, isEdit bool, name, typ, doc, id, errMsg string) {
	tenant := &application.GetTenantOutput{
		ID:       id,
		Name:     name,
		Type:     typ,
		Document: doc,
	}

	c.HTML(http.StatusOK, "tenant/form.html", gin.H{
		"IsEdit": isEdit,
		"Tenant": tenant,
		"Error":  errMsg,
	})
}

func mapDomainError(err error) string {
	switch {
	case errors.Is(err, domain.ErrTenantNameRequired):
		return "Nome é obrigatório"
	case errors.Is(err, domain.ErrTenantDocRequired):
		return "Documento é obrigatório"
	case errors.Is(err, domain.ErrInvalidTenantType):
		return "Tipo inválido: deve ser PF ou PJ"
	case errors.Is(err, domain.ErrTenantDocumentExists):
		return "Documento já cadastrado"
	case errors.Is(err, domain.ErrBlockReasonRequired):
		return "Motivo do bloqueio é obrigatório"
	case errors.Is(err, domain.ErrUnblockReasonRequired):
		return "Motivo do desbloqueio é obrigatório"
	case errors.Is(err, domain.ErrTenantAlreadyBlocked):
		return "Tenant já está bloqueado"
	case errors.Is(err, domain.ErrTenantNotBlocked):
		return "Tenant não está bloqueado"
	case errors.Is(err, domain.ErrReasonTooLong):
		return "Motivo deve ter no máximo 500 caracteres"
	case errors.Is(err, domain.ErrTenantInactive):
		return "Tenant está inativo"
	case errors.Is(err, domain.ErrTenantNotFound):
		return "Tenant não encontrado"
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
