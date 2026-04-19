package domain

import "context"

// TenantBilling é a visão de cobrança de um tenant, retornada pelo
// TenantBillingRepository. Mantém o módulo pagamentos desacoplado do
// agregado Tenant (o repositório lê direto da tabela tenants).
type TenantBilling struct {
	TenantID string
	Active   bool
	Config   BillingConfig
}

// TenantBillingRepository expõe apenas o necessário para o módulo pagamentos
// consultar cobrança dos tenants sem depender do pacote tenant.
type TenantBillingRepository interface {
	GetByID(ctx context.Context, tenantID string) (*TenantBilling, error)
	ListActiveBillable(ctx context.Context) ([]TenantBilling, error)
}
