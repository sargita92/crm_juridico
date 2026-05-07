package domain

// TenantStatusCount — total de tenants por status.
type TenantStatusCount struct {
	Active   int64
	Inactive int64
	Blocked  int64
}

type MonthlyCount struct {
	Label string // "2025-11"
	Count int64
}

type TenantActivity struct {
	TenantID   string
	TenantName string
	LeadCount  int64
}

type InactiveTenant struct {
	TenantID     string
	TenantName   string
	DaysInactive int64
}

type Infrastructure struct {
	APILatencyMs   float64
	Error5xxRate   float64
	ServicesStatus []ServiceStatus
}

type ServiceStatus struct {
	Name string
	Up   bool
}

type SpecialistByTenant struct {
	TenantID   string
	TenantName string
	Total      int64
}

type OverdueTenant struct {
	TenantID   string
	TenantName string
	ValorCents int64
}

type PlanDistribution struct {
	Mensal    int64
	Annual    int64
	Vitalicio int64
	Externo   int64
}

type FinancialBlock struct {
	ReceitaAnoCents    int64
	PendenteTotalCents int64
	AtrasadoTotalCents int64
	PlanDist           PlanDistribution
	TopOverdue         []OverdueTenant // até 10
}

type AdminStats struct {
	Bloco1_Tenants       TenantsBlock
	Bloco2_Uso           UsageBlock
	Bloco3_Health        HealthBlock
	Bloco4_Infra         Infrastructure
	Bloco5_Especialistas SpecialistsBlock
	Bloco6_Financeiro    FinancialBlock
}

type TenantsBlock struct {
	Totals       TenantStatusCount
	NewThisMonth int64
	Last6Months  []MonthlyCount
}

type UsageBlock struct {
	TotalLeads          int64
	TotalMessages       int64
	ActiveConversations int64
}

type HealthBlock struct {
	Top10Active  []TenantActivity
	InactiveList []InactiveTenant
}

type SpecialistsBlock struct {
	Total          int64
	Qualifications int64
	ByTenant       []SpecialistByTenant
}
