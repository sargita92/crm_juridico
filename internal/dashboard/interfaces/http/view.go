package http

import (
	"fmt"
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
	Bloco1_Funil        FunilView
	Bloco2_WhatsApp     WhatsAppView
	Bloco3_Responsaveis []ResponsibleView
	Bloco4_TempoFunil   []ColumnDwellView
	Bloco5_Produtos     []ProductView
	ScopeIsUser         bool
	CurrentUserName     string
	ActiveFunnelName    string
}

type FunilView struct {
	Open          int64
	Won           int64
	Lost          int64
	Total         int64 // open+won+lost
	ColumnTotals  []ColumnLeadsView
	ConversionPct string
	NewToday      int64
	NewThisWeek   int64
}

type ColumnLeadsView struct {
	ColumnName string
	Count      int64
}

type WhatsAppView struct {
	IncomingMessages    int64
	OutgoingMessages    int64
	ActiveConversations int64
	FirstResponseAvg    string // ex: "3.5min" ou "-"
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
	for _, r := range s.Bloco3_Responsaveis {
		pct := 0.0
		if denom := r.Won + r.Lost; denom > 0 {
			pct = float64(r.Won) * 100 / float64(denom)
		}
		out.Bloco3_Responsaveis = append(out.Bloco3_Responsaveis, ResponsibleView{
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
