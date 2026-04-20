package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// Handler orquestra os endpoints HTTP do modulo pagamentos — admin global
// (/admin/payment), aba por tenant (/admin/tenants/:id/payment) e
// portal do tenant (/tenant/payment).
type Handler struct {
	listTenant  *application.ListTenantPayments
	listAll     *application.ListAllPayments
	summary     *application.GetTenantFinancialSummary
	register    *application.RegisterManualPayment
	pay         *application.MarkPaymentAsPaid
	cancel      *application.CancelPayment
	billing     domain.TenantBillingRepository
	paymentRepo domain.PaymentRepository
	portalMw    gin.HandlerFunc
	log         *zap.Logger
}

func NewHandler(
	listTenant *application.ListTenantPayments,
	listAll *application.ListAllPayments,
	summary *application.GetTenantFinancialSummary,
	register *application.RegisterManualPayment,
	pay *application.MarkPaymentAsPaid,
	cancel *application.CancelPayment,
	billing domain.TenantBillingRepository,
	paymentRepo domain.PaymentRepository,
	log *zap.Logger,
) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{
		listTenant:  listTenant,
		listAll:     listAll,
		summary:     summary,
		register:    register,
		pay:         pay,
		cancel:      cancel,
		billing:     billing,
		paymentRepo: paymentRepo,
		log:         log,
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	h.registerAdminRoutes(router, mw)
	h.registerTenantPortalRoutes(router, mw)
}

// === Admin ===

func (h *Handler) AdminListGlobal(c *gin.Context) {
	status, di, df, page := parseFiltersFromQuery(c)
	res, err := h.listAll.Execute(c.Request.Context(), application.ListAllInput{
		TenantID:    c.Query("tenant_id"),
		Status:      status,
		DataInicial: di,
		DataFinal:   df,
		Page:        page,
		PageSize:    domain.DefaultPageSize,
	})
	if err != nil {
		h.log.Error("admin list global failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	items := make([]paymentView, len(res.Items))
	for i, p := range res.Items {
		items[i] = toPaymentView(p)
	}
	c.HTML(http.StatusOK, "pagamentos/admin_list_global.html", gin.H{
		"Items": items, "Total": res.Total, "Page": res.Page,
		"Filters": gin.H{
			"TenantID":    c.Query("tenant_id"),
			"Status":      c.Query("status"),
			"DataInicial": c.Query("data_inicial"),
			"DataFinal":   c.Query("data_final"),
		},
	})
}

func (h *Handler) AdminListTenant(c *gin.Context) {
	tenantID := c.Param("id")
	status, di, df, page := parseFiltersFromQuery(c)
	res, err := h.listTenant.Execute(c.Request.Context(), application.ListTenantInput{
		TenantID: tenantID, Status: status, DataInicial: di, DataFinal: df,
		Page: page, PageSize: domain.DefaultPageSize,
	})
	if err != nil {
		h.log.Error("admin list tenant failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	sum, err := h.summary.Execute(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		h.log.Error("admin tenant summary failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	items := make([]paymentView, len(res.Items))
	for i, p := range res.Items {
		items[i] = toPaymentView(p)
	}
	c.HTML(http.StatusOK, "pagamentos/admin_list_tenant.html", gin.H{
		"TenantID": tenantID,
		"Items":    items,
		"Total":    res.Total,
		"Summary":  toSummaryView(sum),
	})
}

func (h *Handler) AdminFormNovo(c *gin.Context) {
	tenantID := c.Param("id")
	c.HTML(http.StatusOK, "pagamentos/form_novo.html", gin.H{"TenantID": tenantID})
}

func (h *Handler) AdminCreateAvulso(c *gin.Context) {
	tenantID := c.Param("id")
	desc := strings.TrimSpace(c.PostForm("descricao"))
	valorStr := c.PostForm("valor")
	vencStr := c.PostForm("data_vencimento")
	obs := c.PostForm("observacao")

	valor, err := parseValorCents(valorStr)
	if err != nil {
		c.HTML(http.StatusUnprocessableEntity, "pagamentos/form_novo.html", gin.H{
			"TenantID": tenantID, "Error": "Valor invalido",
		})
		return
	}
	venc, err := parseDate(vencStr)
	if err != nil {
		c.HTML(http.StatusUnprocessableEntity, "pagamentos/form_novo.html", gin.H{
			"TenantID": tenantID, "Error": "Data de vencimento invalida",
		})
		return
	}

	_, err = h.register.Execute(c.Request.Context(), application.RegisterManualPaymentInput{
		TenantID:       tenantID,
		Descricao:      desc,
		ValorCents:     valor,
		DataVencimento: venc,
		Observacao:     obs,
	})
	if err != nil {
		if errors.Is(err, domain.ErrDescricaoRequired) || errors.Is(err, domain.ErrValorInvalido) || errors.Is(err, domain.ErrDataVencimentoRequired) {
			c.HTML(http.StatusUnprocessableEntity, "pagamentos/form_novo.html", gin.H{
				"TenantID": tenantID, "Error": err.Error(),
			})
			return
		}
		h.log.Error("create avulso failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	infrastructure.LancadosAvulsoTotal.Inc()
	c.Header("HX-Redirect", "/admin/tenants/"+tenantID+"/payment")
	c.Status(http.StatusOK)
}

func (h *Handler) AdminMarkAsPaid(c *gin.Context) {
	paymentID := c.Param("id")
	userID := currentUserID(c)
	if err := h.pay.Execute(c.Request.Context(), paymentID, userID); err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrInvalidTransition) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
		h.log.Error("mark as paid failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	infrastructure.MarcadosPagoTotal.Inc()
	h.renderRow(c, paymentID)
}

func (h *Handler) AdminCancel(c *gin.Context) {
	paymentID := c.Param("id")
	userID := currentUserID(c)
	motivo := strings.TrimSpace(c.PostForm("motivo"))
	if motivo == "" {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}
	if err := h.cancel.Execute(c.Request.Context(), paymentID, userID, motivo); err != nil {
		if errors.Is(err, domain.ErrMotivoRequired) {
			c.AbortWithStatus(http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		if errors.Is(err, domain.ErrInvalidTransition) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
		h.log.Error("cancel payment failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	infrastructure.CanceladosTotal.Inc()
	h.renderRow(c, paymentID)
}

func (h *Handler) AdminTenantSummary(c *gin.Context) {
	tenantID := c.Param("id")
	sum, err := h.summary.Execute(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrTenantNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "pagamentos/partials/resumo_financeiro.html", gin.H{
		"Summary": toSummaryView(sum),
	})
}

func (h *Handler) renderRow(c *gin.Context, paymentID string) {
	p, err := h.paymentRepo.FindByIDAdmin(c.Request.Context(), paymentID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "pagamentos/partials/payment_row.html", gin.H{
		"Row": toPaymentView(*p),
	})
}

func currentUserID(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
