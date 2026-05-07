package http

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
)

// FormatBRL formata centavos como "R$ 1.234,56" (pt-BR).
func FormatBRL(cents int64) string {
	neg := cents < 0
	if neg {
		cents = -cents
	}
	reais := cents / 100
	centavos := cents % 100
	s := fmt.Sprintf("%d", reais)
	var b strings.Builder
	n := len(s)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%sR$ %s,%02d", sign, b.String(), centavos)
}

// FormatHours formata horas decimais como "X.Xh" ou "Xd Yh" se >= 24h.
func FormatHours(h float64) string {
	if h < 0 {
		h = 0
	}
	if h < 24 {
		return fmt.Sprintf("%.1fh", h)
	}
	days := int(h) / 24
	remH := int(h) % 24
	if remH == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, remH)
}

// FormatPct formata 12.34 como "12.3%". Sem casas se inteiro.
func FormatPct(p float64) string {
	if p == float64(int64(p)) {
		return fmt.Sprintf("%d%%", int64(p))
	}
	return fmt.Sprintf("%.1f%%", p)
}

// TenantView é o view-model do dashboard tenant. Strings já formatadas.
type TenantView struct {
	Bloco1_Funil      FunilView
	Bloco2_WhatsApp   WhatsAppView
	Bloco3_Responsive []ResponsibleView
	Bloco4_TempoFunil []ColumnDwellView
	Bloco5_Produtos   []ProductView
	ScopeIsUser       bool
	CurrentUserName   string
	ActiveFunnelName  string
}

type FunilView struct {
	Open             int64
	Won              int64
	Lost             int64
	Total            int64 // open+won+lost
	ColumnTotals     []ColumnLeadsView
	ColumnTotalsJSON template.JS // JSON array p/ Chart.js (pre-encoded em ToTenantView)
	ConversionPct    string
	NewToday         int64
	NewThisWeek      int64
}

type ColumnLeadsView struct {
	ColumnName string
	Count      int64
}

type WhatsAppView struct {
	IncomingMessages    int64
	OutgoingMessages    int64
	ActiveConversations int64
	FirstResponseAvg    string      // ex: "3.5min" ou "-"
	MessagesJSON        template.JS // JSON {"in":N,"out":M} p/ Chart.js
}

type ResponsibleView struct {
	UserName      string
	Total         int64
	Won           int64
	Lost          int64
	ConversionPct string
}

type ColumnDwellView struct {
	ColumnName     string
	AvgHours       string
	StuckOver7Days int64
}

type ProductView struct {
	ProductName string
	Total       int64
	Won         int64
	Lost        int64
}

func ToTenantView(s *domain.TenantStats) *TenantView {
	if s == nil {
		return &TenantView{}
	}
	out := &TenantView{
		Bloco1_Funil: FunilView{
			Open:          s.Bloco1_Funil.StatusTotals.Open,
			Won:           s.Bloco1_Funil.StatusTotals.Won,
			Lost:          s.Bloco1_Funil.StatusTotals.Lost,
			Total:         s.Bloco1_Funil.StatusTotals.Open + s.Bloco1_Funil.StatusTotals.Won + s.Bloco1_Funil.StatusTotals.Lost,
			ConversionPct: FormatPct(s.Bloco1_Funil.ConversionPct),
			NewToday:      s.Bloco1_Funil.NewToday,
			NewThisWeek:   s.Bloco1_Funil.NewThisWeek,
		},
		Bloco2_WhatsApp: WhatsAppView{
			IncomingMessages:    s.Bloco2_WhatsApp.IncomingMessages,
			OutgoingMessages:    s.Bloco2_WhatsApp.OutgoingMessages,
			ActiveConversations: s.Bloco2_WhatsApp.ActiveConversations,
			FirstResponseAvg:    formatResponseSec(s.Bloco2_WhatsApp.FirstResponseAvgSec),
		},
		ScopeIsUser:      s.ScopeIsUser,
		CurrentUserName:  s.CurrentUserName,
		ActiveFunnelName: s.ActiveFunnelName,
	}
	for _, ct := range s.Bloco1_Funil.ColumnTotals {
		out.Bloco1_Funil.ColumnTotals = append(out.Bloco1_Funil.ColumnTotals, ColumnLeadsView{
			ColumnName: ct.ColumnName, Count: ct.Count,
		})
	}
	for _, r := range s.Bloco3_Responsive {
		pct := 0.0
		if denom := r.Won + r.Lost; denom > 0 {
			pct = float64(r.Won) * 100 / float64(denom)
		}
		out.Bloco3_Responsive = append(out.Bloco3_Responsive, ResponsibleView{
			UserName: r.UserName, Total: r.Total, Won: r.Won, Lost: r.Lost,
			ConversionPct: FormatPct(pct),
		})
	}
	for _, d := range s.Bloco4_TempoFunil {
		out.Bloco4_TempoFunil = append(out.Bloco4_TempoFunil, ColumnDwellView{
			ColumnName:     d.ColumnName,
			AvgHours:       FormatHours(d.AvgHours),
			StuckOver7Days: d.StuckOver7Days,
		})
	}
	for _, p := range s.Bloco5_Produtos {
		out.Bloco5_Produtos = append(out.Bloco5_Produtos, ProductView{
			ProductName: p.ProductName, Total: p.Total, Won: p.Won, Lost: p.Lost,
		})
	}

	// Pré-serializa estruturas que serão consumidas pelo Chart.js no template,
	// usando <script type="application/json"> + JSON.parse no front. Mantém o
	// view-model agnóstico de template/HTML.
	if data, err := json.Marshal(out.Bloco1_Funil.ColumnTotals); err == nil {
		out.Bloco1_Funil.ColumnTotalsJSON = template.JS(data)
	} else {
		out.Bloco1_Funil.ColumnTotalsJSON = template.JS("[]")
	}
	msgData := map[string]int64{
		"in":  s.Bloco2_WhatsApp.IncomingMessages,
		"out": s.Bloco2_WhatsApp.OutgoingMessages,
	}
	if data, err := json.Marshal(msgData); err == nil {
		out.Bloco2_WhatsApp.MessagesJSON = template.JS(data)
	} else {
		out.Bloco2_WhatsApp.MessagesJSON = template.JS("{}")
	}
	return out
}

func formatResponseSec(sec float64) string {
	if sec <= 0 {
		return "-"
	}
	if sec < 60 {
		return fmt.Sprintf("%.0fs", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%.1fmin", sec/60)
	}
	return fmt.Sprintf("%.1fh", sec/3600)
}

// AdminView é o view-model do dashboard admin. Strings já formatadas (BRL, %, etc.).
type AdminView struct {
	Bloco1_Tenants       TenantsView
	Bloco2_Uso           UsageView
	Bloco3_Health        HealthView
	Bloco4_Infra         InfraView
	Bloco5_Especialistas SpecialistsView
	Bloco6_Financeiro    FinancialView
}

type TenantsView struct {
	Active       int64
	Inactive     int64
	Blocked      int64
	Total        int64
	NewThisMonth int64
	Last6Months  []MonthlyView
	MonthlyJSON  template.JS // JSON array de Last6Months p/ Chart.js
}

type MonthlyView struct {
	Label string
	Count int64
}

type UsageView struct {
	TotalLeads          int64
	TotalMessages       int64
	ActiveConversations int64
}

type HealthView struct {
	Top10Active  []TenantActivityView
	InactiveList []InactiveTenantView
	Top10JSON    template.JS // JSON array p/ Chart.js
}

type TenantActivityView struct {
	TenantName string
	LeadCount  int64
}

type InactiveTenantView struct {
	TenantName   string
	DaysInactive int64
}

type InfraView struct {
	APILatencyMs   string // ex: "120.5ms"
	Error5xxRate   string // ex: "0.1%"
	ServicesStatus []ServiceStatusView
}

type ServiceStatusView struct {
	Name string
	Up   bool
}

type SpecialistsView struct {
	Total          int64
	Qualifications int64
	ByTenant       []SpecialistByTenantView
	ByTenantJSON   template.JS // JSON array p/ Chart.js
}

type SpecialistByTenantView struct {
	TenantName string
	Total      int64
}

type FinancialView struct {
	ReceitaAno     string // BRL formatado
	PendenteTotal  string // BRL
	AtrasadoTotal  string // BRL
	PlanDist       PlanDistributionView
	TopOverdue     []OverdueTenantView
	PlanDistJSON   template.JS // JSON {"Mensal":N,"Annual":N,"Vitalicio":N,"Externo":N}
	TopOverdueJSON template.JS // JSON array com TenantName + ValorCents (raw int64)
}

type PlanDistributionView struct {
	Mensal    int64
	Annual    int64
	Vitalicio int64
	Externo   int64
	Total     int64 // soma — útil pra render percentuais no template
}

type OverdueTenantView struct {
	TenantName string
	Valor      string // BRL
	ValorCents int64  // raw para chart (dividido por 100 em JS)
}

func ToAdminView(s *domain.AdminStats) *AdminView {
	if s == nil {
		return &AdminView{}
	}
	out := &AdminView{
		Bloco1_Tenants: TenantsView{
			Active:       s.Bloco1_Tenants.Totals.Active,
			Inactive:     s.Bloco1_Tenants.Totals.Inactive,
			Blocked:      s.Bloco1_Tenants.Totals.Blocked,
			Total:        s.Bloco1_Tenants.Totals.Active + s.Bloco1_Tenants.Totals.Inactive + s.Bloco1_Tenants.Totals.Blocked,
			NewThisMonth: s.Bloco1_Tenants.NewThisMonth,
		},
		Bloco2_Uso: UsageView{
			TotalLeads:          s.Bloco2_Uso.TotalLeads,
			TotalMessages:       s.Bloco2_Uso.TotalMessages,
			ActiveConversations: s.Bloco2_Uso.ActiveConversations,
		},
		Bloco4_Infra: InfraView{
			APILatencyMs: fmt.Sprintf("%.1fms", s.Bloco4_Infra.APILatencyMs),
			Error5xxRate: FormatPct(s.Bloco4_Infra.Error5xxRate * 100),
		},
		Bloco5_Especialistas: SpecialistsView{
			Total:          s.Bloco5_Especialistas.Total,
			Qualifications: s.Bloco5_Especialistas.Qualifications,
		},
		Bloco6_Financeiro: FinancialView{
			ReceitaAno:    FormatBRL(s.Bloco6_Financeiro.ReceitaAnoCents),
			PendenteTotal: FormatBRL(s.Bloco6_Financeiro.PendenteTotalCents),
			AtrasadoTotal: FormatBRL(s.Bloco6_Financeiro.AtrasadoTotalCents),
			PlanDist: PlanDistributionView{
				Mensal:    s.Bloco6_Financeiro.PlanDist.Mensal,
				Annual:    s.Bloco6_Financeiro.PlanDist.Annual,
				Vitalicio: s.Bloco6_Financeiro.PlanDist.Vitalicio,
				Externo:   s.Bloco6_Financeiro.PlanDist.Externo,
				Total: s.Bloco6_Financeiro.PlanDist.Mensal + s.Bloco6_Financeiro.PlanDist.Annual +
					s.Bloco6_Financeiro.PlanDist.Vitalicio + s.Bloco6_Financeiro.PlanDist.Externo,
			},
		},
	}
	for _, m := range s.Bloco1_Tenants.Last6Months {
		out.Bloco1_Tenants.Last6Months = append(out.Bloco1_Tenants.Last6Months, MonthlyView{Label: m.Label, Count: m.Count})
	}
	for _, t := range s.Bloco3_Health.Top10Active {
		out.Bloco3_Health.Top10Active = append(out.Bloco3_Health.Top10Active, TenantActivityView{TenantName: t.TenantName, LeadCount: t.LeadCount})
	}
	for _, t := range s.Bloco3_Health.InactiveList {
		out.Bloco3_Health.InactiveList = append(out.Bloco3_Health.InactiveList, InactiveTenantView{TenantName: t.TenantName, DaysInactive: t.DaysInactive})
	}
	for _, ss := range s.Bloco4_Infra.ServicesStatus {
		out.Bloco4_Infra.ServicesStatus = append(out.Bloco4_Infra.ServicesStatus, ServiceStatusView{Name: ss.Name, Up: ss.Up})
	}
	for _, st := range s.Bloco5_Especialistas.ByTenant {
		out.Bloco5_Especialistas.ByTenant = append(out.Bloco5_Especialistas.ByTenant, SpecialistByTenantView{TenantName: st.TenantName, Total: st.Total})
	}
	for _, ov := range s.Bloco6_Financeiro.TopOverdue {
		out.Bloco6_Financeiro.TopOverdue = append(out.Bloco6_Financeiro.TopOverdue, OverdueTenantView{
			TenantName: ov.TenantName,
			Valor:      FormatBRL(ov.ValorCents),
			ValorCents: ov.ValorCents,
		})
	}

	// Pré-serializa estruturas que serão consumidas pelo Chart.js no template.
	// Usa template.JS para sinalizar a html/template que o conteúdo já é literal
	// JSON seguro (caso contrário ele re-quotaria como string JS dentro de
	// <script type="application/json">).
	if data, err := json.Marshal(out.Bloco1_Tenants.Last6Months); err == nil {
		out.Bloco1_Tenants.MonthlyJSON = template.JS(data)
	} else {
		out.Bloco1_Tenants.MonthlyJSON = template.JS("[]")
	}
	if data, err := json.Marshal(out.Bloco3_Health.Top10Active); err == nil {
		out.Bloco3_Health.Top10JSON = template.JS(data)
	} else {
		out.Bloco3_Health.Top10JSON = template.JS("[]")
	}
	if data, err := json.Marshal(out.Bloco5_Especialistas.ByTenant); err == nil {
		out.Bloco5_Especialistas.ByTenantJSON = template.JS(data)
	} else {
		out.Bloco5_Especialistas.ByTenantJSON = template.JS("[]")
	}
	if data, err := json.Marshal(out.Bloco6_Financeiro.PlanDist); err == nil {
		out.Bloco6_Financeiro.PlanDistJSON = template.JS(data)
	} else {
		out.Bloco6_Financeiro.PlanDistJSON = template.JS("{}")
	}
	if data, err := json.Marshal(out.Bloco6_Financeiro.TopOverdue); err == nil {
		out.Bloco6_Financeiro.TopOverdueJSON = template.JS(data)
	} else {
		out.Bloco6_Financeiro.TopOverdueJSON = template.JS("[]")
	}
	return out
}
