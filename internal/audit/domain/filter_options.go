package domain

import "context"

// TenantSummary e o subset de campos que o dropdown de filtro precisa
// (Tela 1 — F12). Mantemos o struct no dominio audit (e nao importando
// tenant.domain.Tenant) para nao expor campos sensiveis na UI e para
// evitar import cycle entre interfaces/http <-> infrastructure.
type TenantSummary struct {
	ID   string
	Name string
}

// AdminUserSummary segue a mesma logica de TenantSummary — DTO local
// para nao vazar PasswordHash/Status no template e isolar dependencia.
type AdminUserSummary struct {
	ID    string
	Name  string
	Email string
}

// TenantLister popula o dropdown de filtro "Tenant".
//
// Implementacao default fica em internal/audit/infrastructure
// (TenantListerAdapter). Composition root injeta via
// audit.Module.AttachFilters.
type TenantLister interface {
	ListTenants(ctx context.Context) ([]TenantSummary, error)
}

// AdminUserLister popula o dropdown de filtro "Usuario admin".
type AdminUserLister interface {
	ListAdminUsers(ctx context.Context) ([]AdminUserSummary, error)
}
