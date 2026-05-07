package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/funnel/application"
	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

var columnColors = []string{
	"#22c55e", "#3b82f6", "#eab308", "#f97316",
	"#ef4444", "#8b5cf6", "#ec4899", "#6b7280",
}

type Handler struct {
	getKanbanUC      *application.GetKanbanUseCase
	listFunnelsUC    *application.ListFunnelsUseCase
	getFunnelUC      *application.GetFunnelUseCase
	createFunnelUC   *application.CreateFunnelUseCase
	updateFunnelUC   *application.UpdateFunnelUseCase
	toggleFunnelUC   *application.ToggleFunnelUseCase
	createColumnUC   *application.CreateColumnUseCase
	deleteColumnUC   *application.DeleteColumnUseCase
	moveColumnUC     *application.MoveColumnUseCase
	createLeadUC     *application.CreateLeadUseCase
	moveLeadUC       *application.MoveLeadUseCase
	getLeadDetailUC  *application.GetLeadDetailUseCase
	createLeadNoteUC *application.CreateLeadNoteUseCase
	assignLeadUC     *application.AssignLeadUseCase
	leadRepo         domain.LeadRepository
	productLister    domain.ProductLister
	productProvider  domain.ProductProvider
	log              *zap.Logger
}

func NewHandler(
	getKanbanUC *application.GetKanbanUseCase,
	listFunnelsUC *application.ListFunnelsUseCase,
	getFunnelUC *application.GetFunnelUseCase,
	createFunnelUC *application.CreateFunnelUseCase,
	updateFunnelUC *application.UpdateFunnelUseCase,
	toggleFunnelUC *application.ToggleFunnelUseCase,
	createColumnUC *application.CreateColumnUseCase,
	deleteColumnUC *application.DeleteColumnUseCase,
	moveColumnUC *application.MoveColumnUseCase,
	createLeadUC *application.CreateLeadUseCase,
	moveLeadUC *application.MoveLeadUseCase,
	getLeadDetailUC *application.GetLeadDetailUseCase,
	createLeadNoteUC *application.CreateLeadNoteUseCase,
	assignLeadUC *application.AssignLeadUseCase,
	leadRepo domain.LeadRepository,
	productLister domain.ProductLister,
	productProvider domain.ProductProvider,
	log *zap.Logger,
) *Handler {
	return &Handler{
		getKanbanUC:      getKanbanUC,
		listFunnelsUC:    listFunnelsUC,
		getFunnelUC:      getFunnelUC,
		createFunnelUC:   createFunnelUC,
		updateFunnelUC:   updateFunnelUC,
		toggleFunnelUC:   toggleFunnelUC,
		createColumnUC:   createColumnUC,
		deleteColumnUC:   deleteColumnUC,
		moveColumnUC:     moveColumnUC,
		createLeadUC:     createLeadUC,
		moveLeadUC:       moveLeadUC,
		getLeadDetailUC:  getLeadDetailUC,
		createLeadNoteUC: createLeadNoteUC,
		assignLeadUC:     assignLeadUC,
		leadRepo:         leadRepo,
		productLister:    productLister,
		productProvider:  productProvider,
		log:              log,
	}
}

// --- Kanban ---

func (h *Handler) RenderKanbanPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/kanban.html", gin.H{
			"Error":     "Erro ao carregar funis",
			"ActiveNav": "leads",
		})
		return
	}

	var currentFunnelID string
	for _, f := range funnels {
		if f.IsDefault {
			currentFunnelID = f.ID
			break
		}
	}
	if currentFunnelID == "" && len(funnels) > 0 {
		currentFunnelID = funnels[0].ID
	}

	// Load products for filter dropdown
	var products []domain.ProductInfo
	if h.productLister != nil {
		products, _ = h.productLister.ListActiveProducts(c.Request.Context(), tenantID)
	}

	c.HTML(http.StatusOK, "funnel/kanban.html", gin.H{
		"CurrentFunnelID":  currentFunnelID,
		"Funnels":          funnels,
		"Products":         products,
		"CurrentProductID": c.Query("product_id"),
		"ActiveNav":        "leads",
		"OpenLeadID":       h.resolveOpenLeadID(c, tenantID),
	})
}

// resolveOpenLeadID validates that the lead exists and belongs to the tenant.
// On any mismatch/error it returns an empty string — the caller renders the
// page without the drawer loader. We do NOT 404 to avoid a timing oracle
// that would let attackers probe lead IDs across tenants.
func (h *Handler) resolveOpenLeadID(c *gin.Context, tenantID string) string {
	id := c.Query("open")
	if id == "" {
		return ""
	}
	_, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   id,
	})
	if err != nil {
		h.log.Warn("kanban open query ignored",
			zap.String("lead_id", id),
			zap.String("tenant_id", tenantID),
			zap.Error(err),
		)
		return ""
	}
	return id
}

func (h *Handler) RenderKanbanContent(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	search := c.Query("search")
	productID := c.Query("product_id")

	output, err := h.getKanbanUC.Execute(c.Request.Context(), application.GetKanbanInput{
		TenantID:  tenantID,
		FunnelID:  funnelID,
		Search:    search,
		ProductID: productID,
	})
	if err != nil {
		// No funnel = empty kanban (not an error)
		c.HTML(http.StatusOK, "funnel/kanban_content.html", gin.H{})
		return
	}

	// Resolve product names for leads that have a product_id
	h.resolveProductNames(c, output)

	h.log.Info("kanban data",
		zap.String("funnel", output.FunnelName),
		zap.Int("columns", len(output.Columns)),
		zap.Int("leads", output.TotalLeads),
	)

	c.HTML(http.StatusOK, "funnel/kanban_content.html", gin.H{
		"FunnelID":   output.FunnelID,
		"FunnelName": output.FunnelName,
		"Columns":    output.Columns,
		"TotalLeads": output.TotalLeads,
		"Search":     search,
	})
}

// --- Lead Product ---

func (h *Handler) RenderLeadProductForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	var products []domain.ProductInfo
	if h.productLister != nil {
		products, _ = h.productLister.ListActiveProducts(c.Request.Context(), tenantID)
	}

	// Get lead to know current product_id
	lead, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to get lead for product form", zap.Error(err))
		c.HTML(http.StatusNotFound, "funnel/lead_product_form.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	// Find the current product ID from the lead domain object
	domainLead, _ := h.leadRepo.FindByID(c.Request.Context(), leadID)
	var currentProductID string
	if domainLead != nil {
		currentProductID = domainLead.ProductID
	}

	c.HTML(http.StatusOK, "funnel/lead_product_form.html", gin.H{
		"LeadID":           leadID,
		"Products":         products,
		"CurrentProductID": currentProductID,
		"Lead":             lead,
	})
}

func (h *Handler) HandleSetLeadProduct(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")
	productID := c.PostForm("product_id")

	lead, err := h.leadRepo.FindByID(c.Request.Context(), leadID)
	if err != nil {
		h.log.Error("failed to find lead", zap.Error(err))
		c.Status(http.StatusNotFound)
		return
	}
	if lead.TenantID != tenantID {
		c.Status(http.StatusNotFound)
		return
	}

	lead.SetProduct(productID)
	if err := h.leadRepo.Update(c.Request.Context(), lead); err != nil {
		h.log.Error("failed to update lead product", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	// Return updated product section
	var productName string
	if productID != "" && h.productProvider != nil {
		name, err := h.productProvider.FindProductNameByID(c.Request.Context(), productID)
		if err == nil {
			productName = name
		}
	}

	c.HTML(http.StatusOK, "funnel/lead_product_section.html", gin.H{
		"LeadID":      leadID,
		"ProductName": productName,
	})
}

// resolveProductNames enriches KanbanLead with ProductName if available.
func (h *Handler) resolveProductNames(c *gin.Context, output *application.KanbanOutput) {
	if h.productProvider == nil {
		return
	}
	for i := range output.Columns {
		for j := range output.Columns[i].Leads {
			lead := &output.Columns[i].Leads[j]
			if lead.ProductID != "" {
				name, err := h.productProvider.FindProductNameByID(c.Request.Context(), lead.ProductID)
				if err == nil {
					lead.ProductName = name
				}
			}
		}
	}
}

// --- Funnels ---

func (h *Handler) RenderFunnelList(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/funnel_list.html", gin.H{
			"Error":     "Erro ao carregar funis",
			"ActiveNav": "leads",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/funnel_list.html", gin.H{
		"Funnels":   funnels,
		"ActiveNav": "leads",
	})
}

func (h *Handler) RenderFunnelDetail(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	detail, err := h.getFunnelUC.Execute(c.Request.Context(), application.GetFunnelInput{
		TenantID: tenantID,
		FunnelID: funnelID,
	})
	if err != nil {
		h.log.Error("failed to get funnel", zap.Error(err))
		c.HTML(http.StatusNotFound, "funnel/funnel_detail.html", gin.H{
			"Error":     "Funil nao encontrado",
			"ActiveNav": "leads",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/funnel_detail.html", gin.H{
		"Funnel":    detail,
		"ActiveNav": "leads",
	})
}

func (h *Handler) RenderFunnelForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	data := gin.H{
		"ActiveNav": "leads",
	}

	if funnelID != "" {
		detail, err := h.getFunnelUC.Execute(c.Request.Context(), application.GetFunnelInput{
			TenantID: tenantID,
			FunnelID: funnelID,
		})
		if err != nil {
			h.log.Error("failed to get funnel for edit", zap.Error(err))
			c.HTML(http.StatusNotFound, "funnel/funnel_form.html", gin.H{
				"Error": "Funil nao encontrado",
			})
			return
		}
		data["Funnel"] = detail
	}

	c.HTML(http.StatusOK, "funnel/funnel_form.html", data)
}

func (h *Handler) HandleCreateFunnel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	name := c.PostForm("name")
	description := c.PostForm("description")

	_, err := h.createFunnelUC.Execute(c.Request.Context(), application.CreateFunnelInput{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		h.log.Error("failed to create funnel", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "funnel/funnel_form.html", gin.H{
			"Error":       err.Error(),
			"Name":        name,
			"Description": description,
			"ActiveNav":   "leads",
		})
		return
	}

	c.Redirect(http.StatusFound, "/tenant/leads/funnels")
}

func (h *Handler) HandleUpdateFunnel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")
	name := c.PostForm("name")
	description := c.PostForm("description")

	_, err := h.updateFunnelUC.Execute(c.Request.Context(), application.UpdateFunnelInput{
		TenantID:    tenantID,
		FunnelID:    funnelID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		h.log.Error("failed to update funnel", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "funnel/funnel_form.html", gin.H{
			"Error":  "Erro ao atualizar funil: " + err.Error(),
			"Funnel": &application.FunnelDetailOutput{ID: funnelID, Name: name, Description: description},
		})
		return
	}

	c.Header("HX-Redirect", "/tenant/leads/funnels/"+funnelID)
	c.Status(http.StatusOK)
}

func (h *Handler) HandleToggleFunnel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	_, err := h.toggleFunnelUC.Execute(c.Request.Context(), application.ToggleFunnelInput{
		TenantID: tenantID,
		FunnelID: funnelID,
	})
	if err != nil {
		h.log.Error("failed to toggle funnel", zap.Error(err))
	}

	c.Redirect(http.StatusFound, "/tenant/leads/funnels/"+funnelID)
}

// --- Columns ---

func (h *Handler) RenderColumnsSection(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	detail, err := h.getFunnelUC.Execute(c.Request.Context(), application.GetFunnelInput{
		TenantID: tenantID,
		FunnelID: funnelID,
	})
	if err != nil {
		h.log.Error("failed to get funnel columns", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/columns_section.html", gin.H{
			"Error": "Erro ao carregar colunas",
		})
		return
	}

	type columnWithCount struct {
		application.ColumnOutput
		LeadCount int
	}

	cols := make([]columnWithCount, len(detail.Columns))
	for i, col := range detail.Columns {
		count, err := h.leadRepo.CountByColumnID(c.Request.Context(), col.ID)
		if err != nil {
			h.log.Error("failed to count leads for column", zap.String("column_id", col.ID), zap.Error(err))
		}
		cols[i] = columnWithCount{
			ColumnOutput: col,
			LeadCount:    count,
		}
	}

	c.HTML(http.StatusOK, "funnel/columns_section.html", gin.H{
		"FunnelID": funnelID,
		"Columns":  cols,
		"Total":    len(cols),
	})
}

func (h *Handler) RenderColumnForm(c *gin.Context) {
	funnelID := c.Param("id")

	c.HTML(http.StatusOK, "funnel/column_form.html", gin.H{
		"FunnelID": funnelID,
		"Colors":   columnColors,
	})
}

func (h *Handler) HandleCreateColumn(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")
	name := c.PostForm("name")
	colType := c.PostForm("type")
	color := c.PostForm("color")

	_, err := h.createColumnUC.Execute(c.Request.Context(), application.CreateColumnInput{
		TenantID: tenantID,
		FunnelID: funnelID,
		Name:     name,
		Type:     domain.ColumnType(colType),
		Color:    color,
	})
	if err != nil {
		h.log.Error("failed to create column", zap.Error(err))
	}

	h.RenderColumnsSection(c)
}

func (h *Handler) HandleDeleteColumn(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	columnID := c.Param("cid")

	err := h.deleteColumnUC.Execute(c.Request.Context(), application.DeleteColumnInput{
		TenantID: tenantID,
		ColumnID: columnID,
	})
	if err != nil {
		h.log.Error("failed to delete column", zap.Error(err))
	}

	h.RenderColumnsSection(c)
}

func (h *Handler) HandleMoveColumnUp(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")
	columnID := c.Param("cid")

	err := h.moveColumnUC.Execute(c.Request.Context(), application.MoveColumnInput{
		TenantID:  tenantID,
		FunnelID:  funnelID,
		ColumnID:  columnID,
		Direction: "up",
	})
	if err != nil {
		h.log.Error("failed to move column up", zap.Error(err))
	}

	h.RenderColumnsSection(c)
}

func (h *Handler) HandleMoveColumnDown(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")
	columnID := c.Param("cid")

	err := h.moveColumnUC.Execute(c.Request.Context(), application.MoveColumnInput{
		TenantID:  tenantID,
		FunnelID:  funnelID,
		ColumnID:  columnID,
		Direction: "down",
	})
	if err != nil {
		h.log.Error("failed to move column down", zap.Error(err))
	}

	h.RenderColumnsSection(c)
}

// --- Leads ---

func (h *Handler) RenderLeadDetail(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	detail, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to get lead detail", zap.Error(err))
		c.HTML(http.StatusNotFound, "funnel/lead_drawer.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_drawer.html", gin.H{
		"Lead": detail,
	})
}

func (h *Handler) RenderLeadMoveForm(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	lead, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to get lead for move form", zap.Error(err))
		c.HTML(http.StatusNotFound, "funnel/lead_move.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels for move form", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/lead_move.html", gin.H{
			"Error": "Erro ao carregar funis",
		})
		return
	}

	// Get columns for the current funnel
	funnelDetail, err := h.getFunnelUC.Execute(c.Request.Context(), application.GetFunnelInput{
		TenantID: tenantID,
		FunnelID: lead.FunnelID,
	})
	if err != nil {
		h.log.Error("failed to get funnel columns for move form", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/lead_move.html", gin.H{
			"Error": "Erro ao carregar colunas",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_move.html", gin.H{
		"Lead":    lead,
		"Funnels": funnels,
		"Columns": funnelDetail.Columns,
	})
}

func (h *Handler) HandleMoveLead(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")
	columnID := c.PostForm("column_id")
	funnelID := c.PostForm("funnel_id")

	err := h.moveLeadUC.Execute(c.Request.Context(), application.MoveLeadInput{
		TenantID: tenantID,
		LeadID:   leadID,
		ColumnID: columnID,
		FunnelID: funnelID,
	})
	if err != nil {
		h.log.Error("failed to move lead", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "funnel/lead_move.html", gin.H{
			"Error": "Erro ao mover lead: " + err.Error(),
		})
		return
	}

	c.Header("HX-Redirect", "/tenant/leads")
	c.Status(http.StatusOK)
}

func (h *Handler) HandleAssignLead(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")

	var req struct {
		UserID string `json:"user_id" form:"user_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		h.log.Error("invalid assign lead request", zap.Error(err))
		c.Status(http.StatusBadRequest)
		return
	}

	err := h.assignLeadUC.Execute(c.Request.Context(), application.AssignLeadInput{
		LeadID:   leadID,
		UserID:   req.UserID,
		TenantID: tenantID,
	})
	if err != nil {
		h.log.Error("failed to assign lead", zap.Error(err))
		c.Status(http.StatusUnprocessableEntity)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) HandleCreateNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	leadID := c.Param("id")
	content := c.PostForm("content")
	userID := c.GetString("user_id")

	_, err := h.createLeadNoteUC.Execute(c.Request.Context(), application.CreateLeadNoteInput{
		TenantID:  tenantID,
		LeadID:    leadID,
		Content:   content,
		CreatedBy: userID,
	})
	if err != nil {
		h.log.Error("failed to create note", zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "funnel/lead_notes_section.html", gin.H{
			"Error": "Erro ao adicionar anotacao: " + err.Error(),
		})
		return
	}

	// Re-fetch lead detail to get updated notes list
	detail, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   leadID,
	})
	if err != nil {
		h.log.Error("failed to reload lead detail", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_notes_section.html", gin.H{
		"Lead": detail,
	})
}
