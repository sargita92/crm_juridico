package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
)

// TenantList renderiza a listagem read-only de pagamentos para o portal do
// tenant. Nunca expõe acoes (pagar/cancelar) — admin so.
func (h *Handler) TenantList(c *gin.Context) {
	tenantID := currentTenantID(c)
	status, di, df, page := parseFiltersFromQuery(c)
	res, err := h.listTenant.Execute(c.Request.Context(), application.ListTenantInput{
		TenantID:    tenantID,
		Status:      status,
		DataInicial: di,
		DataFinal:   df,
		Page:        page,
		PageSize:    domain.DefaultPageSize,
	})
	if err != nil {
		h.log.Error("tenant list failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	sum, err := h.summary.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("tenant summary failed", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	items := make([]paymentView, len(res.Items))
	for i, p := range res.Items {
		v := toPaymentView(p)
		v.PodeMarcarPago = false
		v.PodeCancelar = false
		items[i] = v
	}
	c.HTML(http.StatusOK, "pagamentos/tenant_list.html", gin.H{
		"Items":   items,
		"Summary": toSummaryView(sum),
		"Total":   res.Total,
	})
}
