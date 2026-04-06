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
	getKanbanUC     *application.GetKanbanUseCase
	listFunnelsUC   *application.ListFunnelsUseCase
	getFunnelUC     *application.GetFunnelUseCase
	createFunnelUC  *application.CreateFunnelUseCase
	updateFunnelUC  *application.UpdateFunnelUseCase
	toggleFunnelUC  *application.ToggleFunnelUseCase
	createColumnUC  *application.CreateColumnUseCase
	deleteColumnUC  *application.DeleteColumnUseCase
	moveColumnUC    *application.MoveColumnUseCase
	createLeadUC    *application.CreateLeadUseCase
	moveLeadUC      *application.MoveLeadUseCase
	getLeadDetailUC *application.GetLeadDetailUseCase
	leadRepo        domain.LeadRepository
	log             *zap.Logger
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
	leadRepo domain.LeadRepository,
	log *zap.Logger,
) *Handler {
	return &Handler{
		getKanbanUC:     getKanbanUC,
		listFunnelsUC:   listFunnelsUC,
		getFunnelUC:     getFunnelUC,
		createFunnelUC:  createFunnelUC,
		updateFunnelUC:  updateFunnelUC,
		toggleFunnelUC:  toggleFunnelUC,
		createColumnUC:  createColumnUC,
		deleteColumnUC:  deleteColumnUC,
		moveColumnUC:    moveColumnUC,
		createLeadUC:    createLeadUC,
		moveLeadUC:      moveLeadUC,
		getLeadDetailUC: getLeadDetailUC,
		leadRepo:        leadRepo,
		log:             log,
	}
}

// --- Kanban ---

func (h *Handler) RenderKanbanPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/kanban.html", gin.H{
			"Error":    "Erro ao carregar funis",
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

	c.HTML(http.StatusOK, "funnel/kanban.html", gin.H{
		"CurrentFunnelID": currentFunnelID,
		"Funnels":         funnels,
		"ActiveNav":       "leads",
	})
}

func (h *Handler) RenderKanbanContent(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Query("funnel_id")
	search := c.Query("search")

	output, err := h.getKanbanUC.Execute(c.Request.Context(), application.GetKanbanInput{
		TenantID: tenantID,
		FunnelID: funnelID,
		Search:   search,
	})
	if err != nil {
		// No funnel = empty kanban (not an error)
		c.HTML(http.StatusOK, "funnel/kanban_content.html", gin.H{})
		return
	}

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

// --- Funnels ---

func (h *Handler) RenderFunnelList(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/funnel_list.html", gin.H{
			"Error":    "Erro ao carregar funis",
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
			"Error":    "Funil nao encontrado",
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
		c.HTML(http.StatusNotFound, "funnel/lead_detail.html", gin.H{
			"Error": "Lead nao encontrado",
		})
		return
	}

	c.HTML(http.StatusOK, "funnel/lead_detail.html", gin.H{
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
