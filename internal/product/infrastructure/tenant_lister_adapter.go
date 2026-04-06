package infrastructure

import (
	"github.com/gin-gonic/gin"

	producthttp "github.com/sasrgita/crm-juridico/internal/product/interfaces/http"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// TenantListerAdapter adapts the tenant repository to the product handler's TenantLister interface.
type TenantListerAdapter struct {
	tenantRepo tenantdomain.TenantRepository
}

func NewTenantListerAdapter(tenantRepo tenantdomain.TenantRepository) *TenantListerAdapter {
	return &TenantListerAdapter{tenantRepo: tenantRepo}
}

func (a *TenantListerAdapter) ListTenants(c *gin.Context) ([]producthttp.TenantInfo, error) {
	tenants, err := a.tenantRepo.FindAll(c.Request.Context())
	if err != nil {
		return nil, err
	}
	result := make([]producthttp.TenantInfo, len(tenants))
	for i, t := range tenants {
		result[i] = producthttp.TenantInfo{
			ID:   t.ID,
			Name: t.Name,
		}
	}
	return result, nil
}
