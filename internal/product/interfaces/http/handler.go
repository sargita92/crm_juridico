package http

import (
	"encoding/csv"
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

// TenantLister provides tenant data for admin views.
type TenantLister interface {
	ListTenants(ctx *gin.Context) ([]TenantInfo, error)
}

// FunnelInfo is a lightweight funnel representation for dropdowns.
type FunnelInfo struct {
	ID   string
	Name string
}

// TenantInfo is a lightweight tenant representation for dropdowns.
type TenantInfo struct {
	ID   string
	Name string
}

type Handler struct {
	createProductUC  *application.CreateProductUseCase
	updateProductUC  *application.UpdateProductUseCase
	listProductsUC   *application.ListProductsUseCase
	listTenantProdUC *application.ListTenantProductsUseCase
	toggleProductUC  *application.ToggleProductUseCase
	deleteProductUC  *application.DeleteProductUseCase
	manageFPUC       *application.ManageFunnelProductsUseCase
	manageTPUC       *application.ManageTenantProductsUseCase
	productRepo      domain.ProductRepository
	fpRepo           domain.FunnelProductRepository
	funnelLister     FunnelLister
	tenantLister     TenantLister
	log              *zap.Logger
}

func NewHandler(
	createProductUC *application.CreateProductUseCase,
	updateProductUC *application.UpdateProductUseCase,
	listProductsUC *application.ListProductsUseCase,
	listTenantProdUC *application.ListTenantProductsUseCase,
	toggleProductUC *application.ToggleProductUseCase,
	deleteProductUC *application.DeleteProductUseCase,
	manageFPUC *application.ManageFunnelProductsUseCase,
	manageTPUC *application.ManageTenantProductsUseCase,
	productRepo domain.ProductRepository,
	fpRepo domain.FunnelProductRepository,
	log *zap.Logger,
) *Handler {
	return &Handler{
		createProductUC:  createProductUC,
		updateProductUC:  updateProductUC,
		listProductsUC:   listProductsUC,
		listTenantProdUC: listTenantProdUC,
		toggleProductUC:  toggleProductUC,
		deleteProductUC:  deleteProductUC,
		manageFPUC:       manageFPUC,
		manageTPUC:       manageTPUC,
		productRepo:      productRepo,
		fpRepo:           fpRepo,
		log:              log,
	}
}

// SetFunnelLister sets the funnel lister dependency (wired after construction to avoid circular deps).
func (h *Handler) SetFunnelLister(fl FunnelLister) {
	h.funnelLister = fl
}

// SetTenantLister sets the tenant lister dependency.
func (h *Handler) SetTenantLister(tl TenantLister) {
	h.tenantLister = tl
}

// ===== ADMIN HANDLERS =====

// RenderAdminProductList renders the admin product list page.
func (h *Handler) RenderAdminProductList(c *gin.Context) {
	products, err := h.listProductsUC.Execute(c.Request.Context(), false)
	if err != nil {
		h.log.Error("failed to list products", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "product/admin_product_list.html", gin.H{
			"Error":     "Erro ao carregar produtos",
			"ActiveNav": "products",
		})
		return
	}

	c.HTML(http.StatusOK, "product/admin_product_list.html", gin.H{
		"Products":  products,
		"ActiveNav": "products",
	})
}

// RenderAdminProductForm renders the admin product create/edit form.
func (h *Handler) RenderAdminProductForm(c *gin.Context) {
	productID := c.Param("id")

	data := gin.H{
		"ActiveNav": "products",
	}

	if productID != "" {
		product, err := h.productRepo.FindByID(c.Request.Context(), productID)
		if err != nil {
			h.log.Error("failed to get product for edit", zap.Error(err))
			c.HTML(http.StatusNotFound, "product/admin_product_form.html", gin.H{
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
				// For admin, we try to list funnels; pass empty tenantID as admin doesn't scope by tenant
				funnels, err := h.funnelLister.ListFunnels(c, "")
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

		// Get tenant associations
		tenantAssocs, err := h.manageTPUC.ListProductTenants(c.Request.Context(), productID)
		if err != nil {
			h.log.Error("failed to get tenant associations", zap.Error(err))
		}

		type tenantAssocDisplay struct {
			TenantID   string
			TenantName string
		}
		var linkedTenants []tenantAssocDisplay
		for _, ta := range tenantAssocs {
			name := ta.TenantID
			if h.tenantLister != nil {
				tenants, err := h.tenantLister.ListTenants(c)
				if err == nil {
					for _, t := range tenants {
						if t.ID == ta.TenantID {
							name = t.Name
							break
						}
					}
				}
			}
			linkedTenants = append(linkedTenants, tenantAssocDisplay{
				TenantID:   ta.TenantID,
				TenantName: name,
			})
		}
		data["LinkedTenants"] = linkedTenants

		// Get available tenants for association
		if h.tenantLister != nil {
			tenants, err := h.tenantLister.ListTenants(c)
			if err == nil {
				data["AvailableTenants"] = tenants
			}
		}
	}

	c.HTML(http.StatusOK, "product/admin_product_form.html", data)
}

// HandleAdminCreateProduct handles product creation (admin).
func (h *Handler) HandleAdminCreateProduct(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	keywordsStr := c.PostForm("keywords")

	keywords := parseKeywords(keywordsStr)

	_, err := h.createProductUC.Execute(c.Request.Context(), application.CreateProductInput{
		Name:        name,
		Description: description,
		Keywords:    keywords,
	})
	if err != nil {
		h.log.Error("failed to create product", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "product/admin_product_form.html", gin.H{
			"Error":       err.Error(),
			"Name":        name,
			"Description": description,
			"Keywords":    keywordsStr,
			"ActiveNav":   "products",
		})
		return
	}

	c.Header("HX-Redirect", "/admin/products")
	c.Status(http.StatusOK)
}

// HandleAdminUpdateProduct handles product update (admin).
func (h *Handler) HandleAdminUpdateProduct(c *gin.Context) {
	productID := c.Param("id")
	name := c.PostForm("name")
	description := c.PostForm("description")
	keywordsStr := c.PostForm("keywords")

	keywords := parseKeywords(keywordsStr)

	_, err := h.updateProductUC.Execute(c.Request.Context(), application.UpdateProductInput{
		ProductID:   productID,
		Name:        name,
		Description: description,
		Keywords:    keywords,
	})
	if err != nil {
		h.log.Error("failed to update product", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "product/admin_product_form.html", gin.H{
			"Error":     "Erro ao atualizar produto: " + err.Error(),
			"Product":   &domain.Product{ID: productID, Name: name, Description: description, Keywords: keywords},
			"Keywords":  keywordsStr,
			"ActiveNav": "products",
		})
		return
	}

	c.Header("HX-Redirect", "/admin/products")
	c.Status(http.StatusOK)
}

// HandleAdminToggleProduct handles product toggle (admin).
func (h *Handler) HandleAdminToggleProduct(c *gin.Context) {
	productID := c.Param("id")

	_, err := h.toggleProductUC.Execute(c.Request.Context(), application.ToggleProductInput{
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to toggle product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/admin/products")
	c.Status(http.StatusOK)
}

// HandleAdminDeleteProduct handles product deletion (admin).
func (h *Handler) HandleAdminDeleteProduct(c *gin.Context) {
	productID := c.Param("id")

	err := h.deleteProductUC.Execute(c.Request.Context(), application.DeleteProductInput{
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to delete product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/admin/products")
	c.Status(http.StatusOK)
}

// HandleAdminLinkFunnel links a funnel to a product (admin).
func (h *Handler) HandleAdminLinkFunnel(c *gin.Context) {
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

	c.Header("HX-Redirect", "/admin/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// HandleAdminUnlinkFunnel unlinks a funnel from a product (admin).
func (h *Handler) HandleAdminUnlinkFunnel(c *gin.Context) {
	productID := c.Param("id")
	funnelID := c.Param("funnelId")

	err := h.manageFPUC.Unlink(c.Request.Context(), application.UnlinkFunnelProductInput{
		FunnelID:  funnelID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to unlink funnel from product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/admin/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// HandleAdminUpdatePriority updates funnel-product priority (admin).
func (h *Handler) HandleAdminUpdatePriority(c *gin.Context) {
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

	c.Header("HX-Redirect", "/admin/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// HandleAdminAssociateTenant associates a product with a tenant (admin).
func (h *Handler) HandleAdminAssociateTenant(c *gin.Context) {
	productID := c.Param("id")
	tenantID := c.PostForm("tenant_id")

	err := h.manageTPUC.Associate(c.Request.Context(), application.AssociateTenantProductInput{
		TenantID:  tenantID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to associate tenant to product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/admin/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// HandleAdminDisassociateTenant removes a tenant-product association (admin).
func (h *Handler) HandleAdminDisassociateTenant(c *gin.Context) {
	productID := c.Param("id")
	tenantID := c.Param("tenantId")

	err := h.manageTPUC.Disassociate(c.Request.Context(), application.DisassociateTenantProductInput{
		TenantID:  tenantID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to disassociate tenant from product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/admin/products/"+productID+"/edit")
	c.Status(http.StatusOK)
}

// ===== TENANT HANDLERS =====

// RenderTenantProductList renders the tenant product list with funnel linking.
func (h *Handler) RenderTenantProductList(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	products, err := h.listTenantProdUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list tenant products", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "product/product_list.html", gin.H{
			"Error":     "Erro ao carregar produtos",
			"ActiveNav": "products",
		})
		return
	}

	// Carrega funnels do tenant uma única vez e monta o índice para resolução
	// do nome de cada vínculo — antes o lookup rodava dentro do laço de produtos.
	var availableFunnels []FunnelInfo
	if h.funnelLister != nil {
		availableFunnels, _ = h.funnelLister.ListFunnels(c, tenantID)
	}
	funnelByID := funnelNameIndex(availableFunnels)

	type funnelLinkDisplay struct {
		FunnelID   string
		FunnelName string
		Priority   int
	}
	type productDisplay struct {
		ID          string
		Name        string
		Description string
		Keywords    []string
		Funnels     []funnelLinkDisplay
	}

	var displayProducts []productDisplay
	for _, p := range products {
		var funnels []funnelLinkDisplay
		for _, fl := range p.Funnels {
			funnels = append(funnels, funnelLinkDisplay{
				FunnelID:   fl.FunnelID,
				FunnelName: resolveFunnelName(fl.FunnelID, funnelByID),
				Priority:   fl.Priority,
			})
		}
		displayProducts = append(displayProducts, productDisplay{
			ID: p.ID, Name: p.Name, Description: p.Description,
			Keywords: p.Keywords, Funnels: funnels,
		})
	}

	c.HTML(http.StatusOK, "product/product_list.html", gin.H{
		"Products":         displayProducts,
		"AvailableFunnels": availableFunnels,
		"ActiveNav":        "products",
	})
}

// HandleTenantLinkFunnel links a funnel to a product (tenant).
func (h *Handler) HandleTenantLinkFunnel(c *gin.Context) {
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

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// HandleTenantUnlinkFunnel unlinks a funnel from a product (tenant).
func (h *Handler) HandleTenantUnlinkFunnel(c *gin.Context) {
	productID := c.Param("id")
	funnelID := c.Param("funnelId")

	err := h.manageFPUC.Unlink(c.Request.Context(), application.UnlinkFunnelProductInput{
		FunnelID:  funnelID,
		ProductID: productID,
	})
	if err != nil {
		h.log.Error("failed to unlink funnel from product", zap.Error(err))
	}

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// HandleTenantUpdatePriority updates funnel-product priority (tenant).
func (h *Handler) HandleTenantUpdatePriority(c *gin.Context) {
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

	c.Header("HX-Redirect", "/tenant/products")
	c.Status(http.StatusOK)
}

// parseKeywords splits a comma-separated input into trimmed keywords.
// Supports quoting so phrases with commas can be passed through, e.g.
//
//	aposentadoria, "preciso revisar, é urgente"
//
// yields ["aposentadoria", "preciso revisar, é urgente"].
func parseKeywords(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	// Pre-normalize: collapse whitespace around quoted fields so that
	// `a,  "x, y"  , b` becomes valid CSV `a, "x, y", b` for encoding/csv,
	// which otherwise refuses bare text after a closing quote.
	normalized := normalizeQuotedCSV(s)
	r := csv.NewReader(strings.NewReader(normalized))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	record, err := r.Read()
	if err != nil {
		// Fallback to legacy simple split so we never reject valid-looking input.
		return legacyParseKeywords(s)
	}
	result := make([]string, 0, len(record))
	for _, p := range record {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// normalizeQuotedCSV trims whitespace immediately after a closing quote (before
// the next comma or end-of-string) so encoding/csv accepts inputs like
// `a, "x, y"  , b` without erroring on "bare \" in non-quoted-field".
func normalizeQuotedCSV(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inQuotes := false
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '"':
			if inQuotes && i+1 < len(s) && s[i+1] == '"' {
				// Escaped quote inside a quoted field.
				b.WriteByte('"')
				b.WriteByte('"')
				i += 2
				continue
			}
			b.WriteByte('"')
			inQuotes = !inQuotes
			if !inQuotes {
				// Just closed a quoted field — skip any trailing whitespace
				// up to the next comma (or end-of-string).
				j := i + 1
				for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
					j++
				}
				i = j
				continue
			}
			i++
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func legacyParseKeywords(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
