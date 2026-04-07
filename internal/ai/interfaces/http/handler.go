package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/ai/application"
	"github.com/sasrgita/crm-juridico/internal/ai/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// ProductLister is a narrow interface to list products for the dropdown.
type ProductLister interface {
	ListAllProducts(ctx context.Context) ([]ProductInfo, error)
}

// ProductInfo holds minimal product data for UI purposes.
type ProductInfo struct {
	ID   string
	Name string
}

// Handler handles HTTP requests for the AI subsystem.
type Handler struct {
	aiConfigRepo      domain.AIConfigRepository
	spProductRepo     domain.SpecialistProductRepository
	stateRepo         domain.ConversationStateRepository
	activateHandoff   *application.ActivateHandoffUseCase
	deactivateHandoff *application.DeactivateHandoffUseCase
	productLister     ProductLister
	log               *zap.Logger
}

// NewHandler creates a new AI HTTP handler.
func NewHandler(
	aiConfigRepo domain.AIConfigRepository,
	spProductRepo domain.SpecialistProductRepository,
	stateRepo domain.ConversationStateRepository,
	activateHandoff *application.ActivateHandoffUseCase,
	deactivateHandoff *application.DeactivateHandoffUseCase,
	productLister ProductLister,
	log *zap.Logger,
) *Handler {
	return &Handler{
		aiConfigRepo:      aiConfigRepo,
		spProductRepo:     spProductRepo,
		stateRepo:         stateRepo,
		activateHandoff:   activateHandoff,
		deactivateHandoff: deactivateHandoff,
		productLister:     productLister,
		log:               log,
	}
}

// RegisterRoutes registers all AI HTTP routes on the provided router.
func (h *Handler) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	// Admin routes
	admin := router.Group("/admin/specialists")
	admin.Use(mw.Auth, mw.Admin)

	admin.GET("/:id/ai-config", h.RenderAIConfig)
	admin.POST("/:id/ai-config", h.HandleSaveAIConfig)
	admin.GET("/:id/products", h.RenderSpecialistProducts)
	admin.POST("/:id/products", h.HandleLinkProduct)
	admin.DELETE("/:id/products/:productId", h.HandleUnlinkProduct)

	// Tenant routes
	tenant := router.Group("/tenant/conversations")
	tenant.Use(mw.Auth, mw.Tenant)

	tenant.POST("/:id/handoff", h.HandleActivateHandoff)
	tenant.POST("/:id/handoff/release", h.HandleDeactivateHandoff)
	tenant.GET("/:id/handoff/status", h.RenderHandoffStatus)
}

// --- Admin handlers ---

// RenderAIConfig renders the AI configuration form for a specialist.
func (h *Handler) RenderAIConfig(c *gin.Context) {
	specialistID := c.Param("id")
	if !validUUID(specialistID) {
		c.HTML(http.StatusBadRequest, "specialist/ai_config_section.html", gin.H{"Error": "ID inválido"})
		return
	}

	config, err := h.aiConfigRepo.FindBySpecialistID(c.Request.Context(), specialistID)
	if err != nil {
		if errors.Is(err, domain.ErrAIConfigNotFound) {
			// Return empty defaults
			config = &domain.AIConfig{
				SpecialistID:    specialistID,
				Provider:        "openai",
				Model:           "gpt-4o-mini",
				Temperature:     0.7,
				MaxTokens:       1024,
				DebounceSeconds: 3,
			}
		} else {
			h.log.Error("ai_handler: find ai config", zap.String("specialist_id", specialistID), zap.Error(err))
			c.HTML(http.StatusInternalServerError, "specialist/ai_config_section.html", gin.H{
				"Error":        "Erro ao carregar configuração",
				"SpecialistID": specialistID,
			})
			return
		}
	}

	c.HTML(http.StatusOK, "specialist/ai_config_section.html", gin.H{
		"SpecialistID": specialistID,
		"Config":       config,
	})
}

// HandleSaveAIConfig saves (create or update) AI configuration for a specialist.
func (h *Handler) HandleSaveAIConfig(c *gin.Context) {
	specialistID := c.Param("id")
	if !validUUID(specialistID) {
		c.HTML(http.StatusBadRequest, "specialist/ai_config_section.html", gin.H{
			"Error":        "ID inválido",
			"SpecialistID": specialistID,
		})
		return
	}

	provider := c.PostForm("provider")
	model := c.PostForm("model")
	temperatureStr := c.PostForm("temperature")
	maxTokensStr := c.PostForm("max_tokens")
	debounceStr := c.PostForm("debounce_seconds")

	temperature, _ := strconv.ParseFloat(temperatureStr, 64)
	maxTokens, _ := strconv.Atoi(maxTokensStr)
	debounceSeconds, _ := strconv.Atoi(debounceStr)

	// Try to find existing config first
	existing, err := h.aiConfigRepo.FindBySpecialistID(c.Request.Context(), specialistID)
	if err != nil && !errors.Is(err, domain.ErrAIConfigNotFound) {
		h.log.Error("ai_handler: find ai config for update", zap.String("specialist_id", specialistID), zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/ai_config_section.html", gin.H{
			"Error":        "Erro ao carregar configuração",
			"SpecialistID": specialistID,
		})
		return
	}

	var config *domain.AIConfig
	if errors.Is(err, domain.ErrAIConfigNotFound) || existing == nil {
		config, err = domain.NewAIConfig(uuid.New().String(), specialistID, provider, model, temperature, maxTokens, debounceSeconds)
		if err != nil {
			c.HTML(http.StatusBadRequest, "specialist/ai_config_section.html", gin.H{
				"Error":        mapAIConfigError(err),
				"SpecialistID": specialistID,
				"Config": &domain.AIConfig{
					SpecialistID:    specialistID,
					Provider:        provider,
					Model:           model,
					Temperature:     temperature,
					MaxTokens:       maxTokens,
					DebounceSeconds: debounceSeconds,
				},
			})
			return
		}
	} else {
		if updateErr := existing.Update(provider, model, temperature, maxTokens, debounceSeconds); updateErr != nil {
			c.HTML(http.StatusBadRequest, "specialist/ai_config_section.html", gin.H{
				"Error":        mapAIConfigError(updateErr),
				"SpecialistID": specialistID,
				"Config":       existing,
			})
			return
		}
		config = existing
	}

	if saveErr := h.aiConfigRepo.CreateOrUpdate(c.Request.Context(), config); saveErr != nil {
		h.log.Error("ai_handler: save ai config", zap.String("specialist_id", specialistID), zap.Error(saveErr))
		c.HTML(http.StatusInternalServerError, "specialist/ai_config_section.html", gin.H{
			"Error":        "Erro ao salvar configuração",
			"SpecialistID": specialistID,
			"Config":       config,
		})
		return
	}

	c.HTML(http.StatusOK, "specialist/ai_config_section.html", gin.H{
		"SpecialistID": specialistID,
		"Config":       config,
		"Success":      "Configuração salva com sucesso",
	})
}

// RenderSpecialistProducts renders the list of products linked to a specialist.
func (h *Handler) RenderSpecialistProducts(c *gin.Context) {
	specialistID := c.Param("id")
	if !validUUID(specialistID) {
		c.HTML(http.StatusBadRequest, "specialist/specialist_products_section.html", gin.H{"Error": "ID inválido"})
		return
	}

	linked, err := h.spProductRepo.FindBySpecialistID(c.Request.Context(), specialistID)
	if err != nil {
		h.log.Error("ai_handler: find specialist products", zap.String("specialist_id", specialistID), zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao carregar produtos",
			"SpecialistID": specialistID,
		})
		return
	}

	allProducts, err := h.productLister.ListAllProducts(c.Request.Context())
	if err != nil {
		h.log.Error("ai_handler: list all products", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao carregar produtos disponíveis",
			"SpecialistID": specialistID,
		})
		return
	}

	// Build set of linked product IDs for dedup
	linkedSet := make(map[string]bool, len(linked))
	for _, lp := range linked {
		linkedSet[lp.ProductID] = true
	}

	// Build available products (not yet linked)
	available := make([]ProductInfo, 0, len(allProducts))
	for _, p := range allProducts {
		if !linkedSet[p.ID] {
			available = append(available, p)
		}
	}

	c.HTML(http.StatusOK, "specialist/specialist_products_section.html", gin.H{
		"SpecialistID":     specialistID,
		"LinkedProducts":   linked,
		"AvailableProducts": available,
	})
}

// HandleLinkProduct links a product to a specialist.
func (h *Handler) HandleLinkProduct(c *gin.Context) {
	specialistID := c.Param("id")
	productID := c.PostForm("product_id")

	if !validUUID(specialistID) {
		c.HTML(http.StatusBadRequest, "specialist/specialist_products_section.html", gin.H{
			"Error":        "ID inválido",
			"SpecialistID": specialistID,
		})
		return
	}

	if productID == "" {
		c.HTML(http.StatusBadRequest, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Selecione um produto",
			"SpecialistID": specialistID,
		})
		return
	}

	// Check if already linked
	exists, err := h.spProductRepo.Exists(c.Request.Context(), specialistID, productID)
	if err != nil {
		h.log.Error("ai_handler: check specialist product exists", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao verificar produto",
			"SpecialistID": specialistID,
		})
		return
	}
	if exists {
		h.renderSpecialistProductsSection(c, specialistID, "Produto já está vinculado", "")
		return
	}

	sp, err := domain.NewSpecialistProduct(uuid.New().String(), specialistID, productID)
	if err != nil {
		c.HTML(http.StatusBadRequest, "specialist/specialist_products_section.html", gin.H{
			"Error":        mapSpecialistProductError(err),
			"SpecialistID": specialistID,
		})
		return
	}

	if err := h.spProductRepo.Create(c.Request.Context(), sp); err != nil {
		h.log.Error("ai_handler: create specialist product", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao vincular produto",
			"SpecialistID": specialistID,
		})
		return
	}

	h.renderSpecialistProductsSection(c, specialistID, "", "Produto vinculado com sucesso")
}

// HandleUnlinkProduct removes a product link from a specialist.
func (h *Handler) HandleUnlinkProduct(c *gin.Context) {
	specialistID := c.Param("id")
	productID := c.Param("productId")

	if err := h.spProductRepo.Delete(c.Request.Context(), specialistID, productID); err != nil {
		if errors.Is(err, domain.ErrSpecialistProductNotFound) {
			h.renderSpecialistProductsSection(c, specialistID, "Produto não estava vinculado", "")
			return
		}
		h.log.Error("ai_handler: delete specialist product", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao desvincular produto",
			"SpecialistID": specialistID,
		})
		return
	}

	h.renderSpecialistProductsSection(c, specialistID, "", "Produto desvinculado com sucesso")
}

// renderSpecialistProductsSection re-renders the products section after an action.
func (h *Handler) renderSpecialistProductsSection(c *gin.Context, specialistID, errMsg, successMsg string) {
	linked, err := h.spProductRepo.FindBySpecialistID(c.Request.Context(), specialistID)
	if err != nil {
		h.log.Error("ai_handler: re-render specialist products", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "specialist/specialist_products_section.html", gin.H{
			"Error":        "Erro ao carregar produtos",
			"SpecialistID": specialistID,
		})
		return
	}

	allProducts, err := h.productLister.ListAllProducts(c.Request.Context())
	if err != nil {
		h.log.Error("ai_handler: list all products for re-render", zap.Error(err))
	}

	linkedSet := make(map[string]bool, len(linked))
	for _, lp := range linked {
		linkedSet[lp.ProductID] = true
	}
	available := make([]ProductInfo, 0, len(allProducts))
	for _, p := range allProducts {
		if !linkedSet[p.ID] {
			available = append(available, p)
		}
	}

	c.HTML(http.StatusOK, "specialist/specialist_products_section.html", gin.H{
		"SpecialistID":      specialistID,
		"LinkedProducts":    linked,
		"AvailableProducts": available,
		"Error":             errMsg,
		"Success":           successMsg,
	})
}

// --- Tenant handlers ---

// HandleActivateHandoff activates human handoff for a conversation.
func (h *Handler) HandleActivateHandoff(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")

	err := h.activateHandoff.Execute(c.Request.Context(), tenantID, convID)
	if err != nil {
		if errors.Is(err, application.ErrHandoffAlreadyActive) {
			h.renderHandoffControls(c, convID, true, "Atendimento manual já está ativo")
			return
		}
		h.log.Error("ai_handler: activate handoff", zap.String("conversation_id", convID), zap.Error(err))
		h.renderHandoffControls(c, convID, false, "Erro ao assumir conversa")
		return
	}

	h.renderHandoffControls(c, convID, true, "")
}

// HandleDeactivateHandoff releases handoff and resumes AI for a conversation.
func (h *Handler) HandleDeactivateHandoff(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")

	err := h.deactivateHandoff.Execute(c.Request.Context(), tenantID, convID)
	if err != nil {
		if errors.Is(err, application.ErrHandoffNotActive) {
			h.renderHandoffControls(c, convID, false, "IA já está ativa")
			return
		}
		h.log.Error("ai_handler: deactivate handoff", zap.String("conversation_id", convID), zap.Error(err))
		h.renderHandoffControls(c, convID, false, "Erro ao devolver para IA")
		return
	}

	h.renderHandoffControls(c, convID, false, "")
}

// RenderHandoffStatus renders the handoff controls partial for a conversation.
func (h *Handler) RenderHandoffStatus(c *gin.Context) {
	convID := c.Param("id")

	state, err := h.stateRepo.FindByConversationID(c.Request.Context(), convID)
	if err != nil {
		if errors.Is(err, domain.ErrConversationStateNotFound) {
			// No state yet means IA is in control
			h.renderHandoffControls(c, convID, false, "")
			return
		}
		h.log.Error("ai_handler: find conversation state", zap.String("conversation_id", convID), zap.Error(err))
		h.renderHandoffControls(c, convID, false, "Erro ao carregar status")
		return
	}

	h.renderHandoffControls(c, convID, state.HandoffActive, "")
}

// renderHandoffControls renders the handoff controls partial with the given state.
func (h *Handler) renderHandoffControls(c *gin.Context, convID string, handoffActive bool, errMsg string) {
	c.HTML(http.StatusOK, "whatsapp/handoff_controls.html", gin.H{
		"ConversationID": convID,
		"HandoffActive":  handoffActive,
		"Error":          errMsg,
	})
}

// validUUID checks that the given string is a valid UUID.
func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// mapAIConfigError maps domain errors to user-friendly messages.
func mapAIConfigError(err error) string {
	switch {
	case errors.Is(err, domain.ErrAIProviderRequired):
		return "Provider é obrigatório"
	case errors.Is(err, domain.ErrModelRequired):
		return "Modelo é obrigatório"
	case errors.Is(err, domain.ErrTemperatureInvalid):
		return "Temperature deve ser entre 0 e 1"
	case errors.Is(err, domain.ErrDebounceInvalid):
		return "Debounce deve ser >= 0"
	default:
		return "Erro interno"
	}
}

// mapSpecialistProductError maps domain errors to user-friendly messages.
func mapSpecialistProductError(err error) string {
	switch {
	case errors.Is(err, domain.ErrSpecialistIDRequired):
		return "Especialista é obrigatório"
	case errors.Is(err, domain.ErrProductIDRequired):
		return "Produto é obrigatório"
	case errors.Is(err, domain.ErrSpecialistProductAlreadyExists):
		return "Produto já está vinculado"
	default:
		return "Erro interno"
	}
}
