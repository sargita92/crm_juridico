package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/product/application"
	"github.com/sasrgita/crm-juridico/internal/product/domain"
)

type Handler struct {
	createProductUC *application.CreateProductUseCase
	updateProductUC *application.UpdateProductUseCase
	listProductsUC  *application.ListProductsUseCase
	toggleProductUC *application.ToggleProductUseCase
	manageFPUC      *application.ManageFunnelProductsUseCase
	productRepo     domain.ProductRepository
	fpRepo          domain.FunnelProductRepository
	log             *zap.Logger
}

func NewHandler(
	createProductUC *application.CreateProductUseCase,
	updateProductUC *application.UpdateProductUseCase,
	listProductsUC *application.ListProductsUseCase,
	toggleProductUC *application.ToggleProductUseCase,
	manageFPUC *application.ManageFunnelProductsUseCase,
	productRepo domain.ProductRepository,
	fpRepo domain.FunnelProductRepository,
	log *zap.Logger,
) *Handler {
	return &Handler{
		createProductUC: createProductUC,
		updateProductUC: updateProductUC,
		listProductsUC:  listProductsUC,
		toggleProductUC: toggleProductUC,
		manageFPUC:      manageFPUC,
		productRepo:     productRepo,
		fpRepo:          fpRepo,
		log:             log,
	}
}

// Stub handlers — will be fully implemented in Task 7
func (h *Handler) RenderProductList(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *Handler) RenderProductForm(c *gin.Context)    { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleCreateProduct(c *gin.Context)  { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleUpdateProduct(c *gin.Context)  { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleToggleProduct(c *gin.Context)  { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleLinkFunnel(c *gin.Context)     { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleUnlinkFunnel(c *gin.Context)   { c.Status(http.StatusNotImplemented) }
func (h *Handler) HandleUpdatePriority(c *gin.Context) { c.Status(http.StatusNotImplemented) }
