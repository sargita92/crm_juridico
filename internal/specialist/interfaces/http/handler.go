package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

type Handler struct {
	createUC        *application.CreateSpecialistUseCase
	listUC          *application.ListSpecialistsUseCase
	getUC           *application.GetSpecialistUseCase
	updateUC        *application.UpdateSpecialistUseCase
	deactivateUC    *application.DeactivateSpecialistUseCase
	activateUC      *application.ActivateSpecialistUseCase
	associateUC     *application.AssociateTenantUseCase
	dissociateUC    *application.DissociateTenantUseCase
	listTenantsUC   *application.ListSpecialistTenantsUseCase
	listAvailableUC *application.ListAvailableTenantsUseCase
	specTenantRepo  domain.SpecialistTenantRepository
}

func NewHandler(
	createUC *application.CreateSpecialistUseCase,
	listUC *application.ListSpecialistsUseCase,
	getUC *application.GetSpecialistUseCase,
	updateUC *application.UpdateSpecialistUseCase,
	deactivateUC *application.DeactivateSpecialistUseCase,
	activateUC *application.ActivateSpecialistUseCase,
	associateUC *application.AssociateTenantUseCase,
	dissociateUC *application.DissociateTenantUseCase,
	listTenantsUC *application.ListSpecialistTenantsUseCase,
	listAvailableUC *application.ListAvailableTenantsUseCase,
	specTenantRepo domain.SpecialistTenantRepository,
) *Handler {
	return &Handler{
		createUC:        createUC,
		listUC:          listUC,
		getUC:           getUC,
		updateUC:        updateUC,
		deactivateUC:    deactivateUC,
		activateUC:      activateUC,
		associateUC:     associateUC,
		dissociateUC:    dissociateUC,
		listTenantsUC:   listTenantsUC,
		listAvailableUC: listAvailableUC,
		specTenantRepo:  specTenantRepo,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, adminMw gin.HandlerFunc) {
	admin := router.Group("/admin/specialists")
	admin.Use(authMw, adminMw)

	admin.GET("", h.RenderList)
	admin.GET("/new", h.RenderCreateForm)
	admin.POST("", h.HandleCreate)
	admin.GET("/:id", h.RenderDetail)
	admin.GET("/:id/edit", h.RenderEditForm)
	admin.PUT("/:id", h.HandleUpdate)
	admin.POST("/:id/deactivate", h.HandleDeactivate)
	admin.POST("/:id/activate", h.HandleActivate)
	admin.GET("/:id/tenants", h.HandleListTenants)
	admin.GET("/:id/tenants/available", h.HandleListAvailable)
	admin.POST("/:id/tenants", h.HandleAssociateTenants)
	admin.DELETE("/:id/tenants/:tenant_id", h.HandleDissociateTenant)
	admin.POST("/:id/tenants/:tenant_id/default", h.HandleSetDefault)
}

func (h *Handler) RenderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	input := application.ListSpecialistsInput{
		Search: c.Query("search"),
		Status: c.Query("status"),
		Page:   page,
		Limit:  limit,
	}

	output, err := h.listUC.Execute(c.Request.Context(), input)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "specialist/list.html", gin.H{"Error": "Erro ao carregar especialistas"})
		return
	}

	data := gin.H{
		"Specialists": output.Specialists,
		"Total":       output.Total,
		"Page":        output.Page,
		"Limit":       output.Limit,
		"Search":      input.Search,
		"Status":      input.Status,
		"HasPrev":     output.Page > 1,
		"HasNext":     int64(output.Page*output.Limit) < output.Total,
		"PrevPage":    output.Page - 1,
		"NextPage":    output.Page + 1,
		"StartItem":   (output.Page-1)*output.Limit + 1,
		"EndItem":     min(int64(output.Page*output.Limit), output.Total),
	}

	if c.GetHeader("HX-Request") == "true" {
		c.HTML(http.StatusOK, "specialist/table.html", data)
		return
	}
	c.HTML(http.StatusOK, "specialist/list.html", data)
}

func (h *Handler) RenderCreateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "specialist/form.html", gin.H{
		"IsEdit":     false,
		"Specialist": nil,
		"Error":      "",
	})
}

func (h *Handler) HandleCreate(c *gin.Context) {
	input := application.CreateSpecialistInput{
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		Prompt:      c.PostForm("prompt"),
	}

	output, err := h.createUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.renderFormError(c, false, input.Name, input.Description, input.Prompt, "", mapDomainError(err))
		return
	}

	c.Header("HX-Redirect", "/admin/specialists/"+output.ID)
	c.Status(http.StatusOK)
}

func (h *Handler) RenderDetail(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.HTML(http.StatusBadRequest, "specialist/detail.html", gin.H{"Error": "ID inválido"})
		return
	}

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.HTML(http.StatusNotFound, "specialist/detail.html", gin.H{"Error": "Especialista não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "specialist/detail.html", gin.H{"Error": "Erro ao carregar especialista"})
		return
	}

	c.HTML(http.StatusOK, "specialist/detail.html", gin.H{
		"Specialist": output,
		"Error":      "",
	})
}

func (h *Handler) RenderEditForm(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.HTML(http.StatusBadRequest, "specialist/form.html", gin.H{"Error": "ID inválido"})
		return
	}

	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.HTML(http.StatusNotFound, "specialist/form.html", gin.H{"Error": "Especialista não encontrado"})
			return
		}
		c.HTML(http.StatusInternalServerError, "specialist/form.html", gin.H{"Error": "Erro ao carregar especialista"})
		return
	}

	c.HTML(http.StatusOK, "specialist/form.html", gin.H{
		"IsEdit":     true,
		"Specialist": output,
		"Error":      "",
	})
}

func (h *Handler) HandleUpdate(c *gin.Context) {
	id := c.Param("id")

	input := application.UpdateSpecialistInput{
		ID:          id,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		Prompt:      c.PostForm("prompt"),
	}

	_, err := h.updateUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.renderFormError(c, true, input.Name, input.Description, input.Prompt, id, mapDomainError(err))
		return
	}

	c.Header("HX-Redirect", "/admin/specialists/"+id)
	c.Status(http.StatusOK)
}

func (h *Handler) HandleDeactivate(c *gin.Context) {
	id := c.Param("id")

	err := h.deactivateUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.renderDetailAfterAction(c, id, "Especialista desativado com sucesso")
}

func (h *Handler) HandleActivate(c *gin.Context) {
	id := c.Param("id")

	err := h.activateUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	h.renderDetailAfterAction(c, id, "Especialista reativado com sucesso")
}

func (h *Handler) HandleListTenants(c *gin.Context) {
	id := c.Param("id")

	items, err := h.listTenantsUC.Execute(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar tenants"})
		return
	}

	c.HTML(http.StatusOK, "specialist/tenants_section.html", gin.H{
		"Tenants":      items,
		"SpecialistID": id,
	})
}

func (h *Handler) HandleListAvailable(c *gin.Context) {
	id := c.Param("id")
	search := c.Query("search")

	items, err := h.listAvailableUC.Execute(c.Request.Context(), id, search)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar tenants disponíveis"})
		return
	}

	c.HTML(http.StatusOK, "specialist/tenants_modal.html", gin.H{
		"AvailableTenants": items,
		"SpecialistID":     id,
		"Search":           search,
	})
}

func (h *Handler) HandleAssociateTenants(c *gin.Context) {
	id := c.Param("id")
	tenantIDs := c.PostFormArray("tenant_ids[]")

	if len(tenantIDs) == 0 {
		tenantIDs = c.PostFormArray("tenant_ids")
	}

	if len(tenantIDs) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Selecione pelo menos um tenant"})
		return
	}
	if len(tenantIDs) > 50 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Máximo de 50 tenants por associação"})
		return
	}

	err := h.associateUC.Execute(c.Request.Context(), application.AssociateTenantInput{
		SpecialistID: id,
		TenantIDs:    tenantIDs,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, domain.ErrSpecialistInactive) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Especialista está inativo"})
			return
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": mapDomainError(err)})
		return
	}

	// Re-render tenants section
	h.HandleListTenants(c)
}

func (h *Handler) HandleDissociateTenant(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.Param("tenant_id")

	err := h.dissociateUC.Execute(c.Request.Context(), application.DissociateTenantInput{
		SpecialistID: id,
		TenantID:     tenantID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSpecialistNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Especialista não encontrado"})
			return
		}
		if errors.Is(err, domain.ErrTenantNotAssociated) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Tenant não está associado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao desassociar tenant"})
		return
	}

	// Re-render tenants section
	h.HandleListTenants(c)
}

func (h *Handler) HandleSetDefault(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.Param("tenant_id")

	if err := h.specTenantRepo.SetDefault(c.Request.Context(), id, tenantID); err != nil {
		if errors.Is(err, domain.ErrTenantNotAssociated) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Tenant não está associado"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao definir default"})
		return
	}

	// Re-render tenants section
	h.HandleListTenants(c)
}

func (h *Handler) renderDetailAfterAction(c *gin.Context, id, successMsg string) {
	output, err := h.getUC.Execute(c.Request.Context(), id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Erro ao carregar especialista"})
		return
	}

	c.HTML(http.StatusOK, "specialist/detail.html", gin.H{
		"Specialist": output,
		"Success":    successMsg,
		"Error":      "",
	})
}

func (h *Handler) renderFormError(c *gin.Context, isEdit bool, name, description, prompt, id, errMsg string) {
	specialist := &application.GetSpecialistOutput{
		ID:          id,
		Name:        name,
		Description: description,
		Prompt:      prompt,
	}

	c.HTML(http.StatusOK, "specialist/form.html", gin.H{
		"IsEdit":     isEdit,
		"Specialist": specialist,
		"Error":      errMsg,
	})
}

func mapDomainError(err error) string {
	switch {
	case errors.Is(err, domain.ErrSpecialistNameRequired):
		return "Nome é obrigatório"
	case errors.Is(err, domain.ErrSpecialistPromptRequired):
		return "Prompt é obrigatório"
	case errors.Is(err, domain.ErrPromptTooLong):
		return "Prompt excede o tamanho máximo"
	case errors.Is(err, domain.ErrDescriptionTooLong):
		return "Descrição excede o tamanho máximo"
	case errors.Is(err, domain.ErrSpecialistAlreadyInactive):
		return "Especialista já está inativo"
	case errors.Is(err, domain.ErrSpecialistAlreadyActive):
		return "Especialista já está ativo"
	case errors.Is(err, domain.ErrSpecialistInactive):
		return "Especialista está inativo"
	case errors.Is(err, domain.ErrTenantAlreadyAssociated):
		return "Tenant já está associado"
	case errors.Is(err, domain.ErrTenantNotAssociated):
		return "Tenant não está associado"
	case errors.Is(err, domain.ErrSpecialistNotFound):
		return "Especialista não encontrado"
	case errors.Is(err, tenantdomain.ErrTenantNotFound):
		return "Tenant não encontrado"
	case errors.Is(err, tenantdomain.ErrTenantInactive):
		return "Tenant está inativo"
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
