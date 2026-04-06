package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/product/application"
	"github.com/sasrgita/crm-juridico/internal/product/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// FunnelLister provides funnel data to the product handler for the link form.
type FunnelLister interface {
	ListFunnels(ctx *gin.Context, tenantID string) ([]FunnelInfo, error)
}

// FunnelInfo is a lightweight funnel representation for dropdowns.
type FunnelInfo struct {
	ID   string
	Name string
}

type Handler struct {
	createProductUC *application.CreateProductUseCase
	updateProductUC *application.UpdateProductUseCase
	listProductsUC  *application.ListProductsUseCase
	toggleProductUC *application.ToggleProductUseCase
	manageFPUC      *application.ManageFunnelProductsUseCase
	productRepo     domain.ProductRepository
	fpRepo          domain.FunnelProductRepository
	funnelLister    FunnelLister
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

// SetFunnelLister sets the funnel lister dependency (wired after construction to avoid circular deps).
func (h *Handler) SetFunnelLister(fl FunnelLister) {
	h.funnelLister = fl
}

// --- Product List ---

func (h *Handler) RenderProductList(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	products, err := h.listProductsUC.Execute(c.Request.Context(), tenantID, false)
	if err != nil {
		h.log.Error("failed to list products", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "product/product_list.html", gin.H{
			"Error":     "Erro ao carregar produtos",
			"ActiveNav": "products",
		})
		return
	}

	c.HTML(http.StatusOK, "product/product_list.html", gin.H{
		"Products":  products,
		"ActiveNav": "products",
	})
}

// --- Product Form ---

func (h *Handler) RenderProductForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	productID := c.Param("id")

	data := gin.H{
		"ActiveNav": "products",
	}

	if productID != "" {
		product, err := h.productRepo.FindByID(c.Request.Context(), productID)
		if err != nil {
			h.log.Error("failed to get product for edit", zap.Error(err))
			c.HTML(http.StatusNotFound, "product/product_form.html", gin.H{
				"Error":     "Produto nao encontrado",
				"ActiveNav": "products",
			})
			return
		}
		if product.TenantID != tenantID {
			c.HTML(http.StatusNotFound, "product/product_form.html", gin.H{
				"Error":     "Produto nao encontrado",
				"ActiveNav": "products",
			})
			return
		}

		// Get linked funnels
		funnelLinks, err := h.fpRepo.FindByProductID(c.Request.Context(), productID)
		if err != nil {
			h.log.Error("failed to get funnel links", zap.Error(err))
		}

		type funnelLinkDisplay struct {
			FunnelID   string
			FunnelName string
			Priority   int
		}

		var linkedFunnels []funnelLinkDisplay
		for _, fl := range funnelLinks {
			name := fl.FunnelID // fallback to ID
			if h.funnelLister != nil {
				funnels, err := h.funnelLister.ListFunnels(c, tenantID)
				if err == nil {
					for _, f := range funnels {
						if f.ID == fl.FunnelID {
							name = f.Name
							break
						}
					}
				}
			}
			linkedFunnels = append(linkedFunnels, funnelLinkDisplay{
				FunnelID:   fl.FunnelID,
				FunnelName: name,
				Priority:   fl.Priority,
			})
		}

		data["Product"] = product
		data["Keywords"] = strings.Join(product.Keywords, ", ")
		data["LinkedFunnels"] = linkedFunnels

		// Get available funnels for linking
		if h.funnelLister != nil {
			funnels, err := h.funnelLister.ListFunnels(c, tenantID)
			if err == nil {
				data["AvailableFunnels"] = funnels
			}
		}
	}

	c.HTML(http.StatusOK, "product/product_form.html", data)
}

// --- Create Product ---

func (h *Handler) HandleCreateProduct(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	name := c.PostForm("name")
	description := c.PostForm("description")
	keywordsStr := c.PostForm("keywords")

	keywords := parseKeywords(keywordsStr)

	_, err := h.createProductUC.Execute(c.Request.Context(), application.CreateProductInput{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Keywords:    keywords,
	})
	if err != nil {
		h.log.Error("failed to create product", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "product/product_form.html", gin.H{
			"Error":       err.Error(),
			"Name":        name,
			"Description": description,
			"Keywords":    keywordsStr,
			"ActiveNav":   "products",
		})
		return
	}

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// --- Update Product ---

func (h *Handler) HandleUpdateProduct(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	productID := c.Param("id")
	name := c.PostForm("name")
	description := c.PostForm("description")
	keywordsStr := c.PostForm("keywords")

	keywords := parseKeywords(keywordsStr)

	_, err := h.updateProductUC.Execute(c.Request.Context(), application.UpdateProductInput{
		TenantID:    tenantID,
		ProductID:   productID,
		Name:        name,
		Description: description,
		Keywords:    keywords,
	})
	if err != nil {
		h.log.Error("failed to update product", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "product/product_form.html", gin.H{
			"Error":     "Erro ao atualizar produto: " + err.Error(),
			"Product":   &domain.Product{ID: productID, Name: name, Description: description, Keywords: keywords},
			"Keywords":  keywordsStr,
			"ActiveNav": "products",
		})
		return
	}

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// --- Toggle Product ---

func (h *Handler) HandleToggleProduct(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	productID := c.Param("id")

	_, err := h.toggleProductUC.Execute(c.Request.Context(), application.ToggleProductInput{
		TenantID:  tenantID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to toggle product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// --- Link Funnel ---

func (h *Handler) HandleLinkFunnel(c *gin.Context) {
	productID := c.Param("id")
	funnelID := c.PostForm("funnel_id")
	priorityStr := c.PostForm("priority")

	priority, err := strconv.Atoi(priorityStr)
	if err != nil || priority < 1 {
		priority = 1
	}

	err = h.manageFPUC.Link(c.Request.Context(), application.LinkFunnelProductInput{
		FunnelID:  funnelID,
		ProductID: productID,
		Priority:  priority,
	})
	if err != nil {
		h.log.Error("failed to link funnel to product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/tenant/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// --- Unlink Funnel ---

func (h *Handler) HandleUnlinkFunnel(c *gin.Context) {
	productID := c.Param("id")
	funnelID := c.Param("funnelId")

	err := h.manageFPUC.Unlink(c.Request.Context(), application.UnlinkFunnelProductInput{
		FunnelID:  funnelID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to unlink funnel from product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/tenant/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// --- Update Priority ---

func (h *Handler) HandleUpdatePriority(c *gin.Context) {
	productID := c.Param("id")
	funnelID := c.Param("funnelId")
	priorityStr := c.PostForm("priority")

	priority, err := strconv.Atoi(priorityStr)
	if err != nil || priority < 1 {
		priority = 1
	}

	err = h.manageFPUC.UpdatePriority(c.Request.Context(), application.UpdateFunnelProductPriorityInput{
		FunnelID:  funnelID,
		ProductID: productID,
		Priority:  priority,
	})
	if err != nil {
		h.log.Error("failed to update priority", zap.Error(err))
	}

	c.Header("HX-Redirect", "/tenant/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// parseKeywords splits a comma-separated string into trimmed non-empty keywords.
func parseKeywords(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
