package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/automation/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// Handler holds the automation CRUD use case and provides HTTP handlers.
type Handler struct {
	crudUC *application.CRUDUseCase
	log    *zap.Logger
}

// NewHandler constructs an automation Handler.
func NewHandler(crudUC *application.CRUDUseCase, log *zap.Logger) *Handler {
	return &Handler{crudUC: crudUC, log: log}
}

// ListByFunnel handles GET /tenant/leads/funnels/:id/automations.
func (h *Handler) ListByFunnel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	out, err := h.crudUC.ListByFunnel(c.Request.Context(), tenantID, funnelID)
	if err != nil {
		h.log.Error("failed to list automations", zap.String("funnel_id", funnelID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list automations"})
		return
	}

	c.JSON(http.StatusOK, out)
}

// Create handles POST /tenant/leads/funnels/:id/automations.
func (h *Handler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	funnelID := c.Param("id")

	var req struct {
		ColumnID string                 `json:"column_id"`
		Type     string                 `json:"type"`
		Config   map[string]interface{} `json:"config"`
		Priority int                    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.crudUC.Create(c.Request.Context(), application.CreateAutomationInput{
		TenantID: tenantID,
		FunnelID: funnelID,
		ColumnID: req.ColumnID,
		Type:     req.Type,
		Config:   req.Config,
		Priority: req.Priority,
	})
	if err != nil {
		h.log.Error("failed to create automation", zap.Error(err))
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, out)
}

// GetDetail handles GET /tenant/leads/automations/:id.
func (h *Handler) GetDetail(c *gin.Context) {
	id := c.Param("id")

	out, err := h.crudUC.GetByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get automation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "automation not found"})
		return
	}

	c.JSON(http.StatusOK, out)
}

// Update handles PUT /tenant/leads/automations/:id.
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ColumnID string                 `json:"column_id"`
		Config   map[string]interface{} `json:"config"`
		Priority int                    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.crudUC.Update(c.Request.Context(), id, application.CreateAutomationInput{
		ColumnID: req.ColumnID,
		Config:   req.Config,
		Priority: req.Priority,
	})
	if err != nil {
		h.log.Error("failed to update automation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, out)
}

// Delete handles DELETE /tenant/leads/automations/:id.
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.crudUC.Delete(c.Request.Context(), id); err != nil {
		h.log.Error("failed to delete automation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "automation not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

// Toggle handles POST /tenant/leads/automations/:id/toggle.
func (h *Handler) Toggle(c *gin.Context) {
	id := c.Param("id")

	if err := h.crudUC.Toggle(c.Request.Context(), id); err != nil {
		h.log.Error("failed to toggle automation", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "automation not found"})
		return
	}

	c.Status(http.StatusOK)
}

// GetLogs handles GET /tenant/leads/automations/:id/logs.
func (h *Handler) GetLogs(c *gin.Context) {
	id := c.Param("id")

	limit := 20
	offset := 0
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	out, err := h.crudUC.GetLogs(c.Request.Context(), id, limit, offset)
	if err != nil {
		h.log.Error("failed to get automation logs", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get logs"})
		return
	}

	c.JSON(http.StatusOK, out)
}
