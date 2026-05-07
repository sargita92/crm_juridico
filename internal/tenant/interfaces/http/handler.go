package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	pagdomain "github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
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
		"Tenants":   output.Tenants,
		"Total":     output.Total,
		"Page":      output.Page,
		"Limit":     output.Limit,
		"Search":    input.Search,
		"Status":    input.Status,
		"Type":      input.Type,
		"HasPrev":   output.Page > 1,
		"HasNext":   int64(output.Page*output.Limit) < output.Total,
		"PrevPage":  output.Page - 1,
		"NextPage":  output.Page + 1,
		"StartItem": (output.Page-1)*output.Limit + 1,
		"EndItem":   min(int64(output.Page*output.Limit), output.Total),
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
		h.renderFormError(c, false, &application.GetTenantOutput{
			Name:     input.Name,
			Type:     input.Type,
			Document: input.Document,
		}, mapDomainError(err))
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

	name := c.PostForm("name")
	typ := c.PostForm("type")
	doc := c.PostForm("document")

	billing, formTenant, billErr := parseBillingForm(c, id, name, typ, doc)
	if billErr != "" {
		h.renderFormError(c, true, formTenant, billErr)
		return
	}

	input := application.UpdateTenantInput{
		ID:       id,
		Name:     name,
		Type:     typ,
		Document: doc,
		Billing:  billing,
	}

	_, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.renderFormError(c, true, formTenant, mapDomainError(err))
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

func (h *Handler) renderFormError(c *gin.Context, isEdit bool, tenant *application.GetTenantOutput, errMsg string) {
	c.HTML(http.StatusOK, "tenant/form.html", gin.H{
		"IsEdit": isEdit,
		"Tenant": tenant,
		"Error":  errMsg,
	})
}

// parseBillingForm reads the billing fields from the POST form. It returns the
// UpdateTenantBilling pointer (nil if plano is empty), a GetTenantOutput that
// mirrors the submitted values for re-rendering the form, and a non-empty
// errMsg when the raw inputs cannot be parsed.
func parseBillingForm(c *gin.Context, id, name, typ, doc string) (*application.UpdateTenantBilling, *application.GetTenantOutput, string) {
	plano := c.PostForm("plano")
	valorStr := c.PostForm("valor_cobranca")
	diaStr := c.PostForm("dia_vencimento")
	dataStr := c.PostForm("data_inicio_cobranca")
	exibir := c.PostForm("exibir_pagamentos") == "true"

	formTenant := &application.GetTenantOutput{
		ID: id, Name: name, Type: typ, Document: doc,
		Plano: plano, ExibirPagamentos: exibir,
	}

	if plano == "" {
		return nil, formTenant, ""
	}

	var valorCents *int64
	if valorStr != "" && (plano == "mensal" || plano == "annual") {
		f, err := strconv.ParseFloat(strings.ReplaceAll(valorStr, ",", "."), 64)
		if err != nil {
			return nil, formTenant, "Valor de cobrança inválido"
		}
		v := int64(f * 100)
		valorCents = &v
		formTenant.ValorCobrancaCents = valorCents
	}

	var dia *uint8
	if diaStr != "" && (plano == "mensal" || plano == "annual") {
		n, err := strconv.Atoi(diaStr)
		if err != nil || n < 1 || n > 28 {
			return nil, formTenant, "Dia de vencimento deve estar entre 1 e 28"
		}
		u := uint8(n)
		dia = &u
		formTenant.DiaVencimento = dia
	}

	var dataInicio *time.Time
	if dataStr != "" && (plano == "mensal" || plano == "annual") {
		t, err := time.Parse("2006-01-02", dataStr)
		if err != nil {
			return nil, formTenant, "Data de início inválida"
		}
		dataInicio = &t
		formTenant.DataInicioCobranca = dataInicio
	}

	return &application.UpdateTenantBilling{
		Plano:              plano,
		ValorCobrancaCents: valorCents,
		DiaVencimento:      dia,
		DataInicioCobranca: dataInicio,
		ExibirPagamentos:   exibir,
	}, formTenant, ""
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
	case errors.Is(err, pagdomain.ErrInvalidPlano):
		return "Configuração de cobrança inválida: plano mensal/annual requer valor, dia (1-28) e data de início"
	case errors.Is(err, pagdomain.ErrValorInvalido):
		return "Valor de cobrança deve ser maior que zero"
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
