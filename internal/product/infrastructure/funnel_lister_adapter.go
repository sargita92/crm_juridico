package infrastructure

import (
	"github.com/gin-gonic/gin"

	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	producthttp "github.com/sasrgita/crm-juridico/internal/product/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// FunnelListerAdapter adapts the funnel ListFunnelsUseCase to the product handler's FunnelLister interface.
type FunnelListerAdapter struct {
	listFunnelsUC *funnelapp.ListFunnelsUseCase
}

func NewFunnelListerAdapter(listFunnelsUC *funnelapp.ListFunnelsUseCase) *FunnelListerAdapter {
	return &FunnelListerAdapter{listFunnelsUC: listFunnelsUC}
}

func (a *FunnelListerAdapter) ListFunnels(c *gin.Context, tenantID string) ([]producthttp.FunnelInfo, error) {
	_ = middleware.GetTenantID(c.Request.Context()) // ensure context is valid
	funnels, err := a.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]producthttp.FunnelInfo, len(funnels))
	for i, f := range funnels {
		result[i] = producthttp.FunnelInfo{
			ID:   f.ID,
			Name: f.Name,
		}
	}
	return result, nil
}
