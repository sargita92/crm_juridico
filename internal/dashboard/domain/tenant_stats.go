package domain

// LeadStatusCount é um contador por status do lead.
type LeadStatusCount struct {
	Open int64
	Won  int64
	Lost int64
}

// ColumnLeadsCount — leads por coluna do funil ativo.
type ColumnLeadsCount struct {
	ColumnID   string
	ColumnName string
	OrderIndex int
	Count      int64
}

// ResponsiblePerformance — leads por responsável e convertsão.
type ResponsiblePerformance struct {
	UserID   string
	UserName string
	Total    int64
	Won      int64
	Lost     int64
}

// ColumnDwell — tempo médio de permanência na coluna (em horas) e leads parados > 7 dias.
type ColumnDwell struct {
	ColumnID       string
	ColumnName     string
	OrderIndex     int
	AvgHours       float64
	StuckOver7Days int64
}

// ProductLeadsCount — leads por produto.
type ProductLeadsCount struct {
	ProductID   string
	ProductName string
	Total       int64
	Won         int64
	Lost        int64
}

// WhatsAppStats — agregados de conversa/mensagem do tenant.
type WhatsAppStats struct {
	IncomingMessages    int64
	OutgoingMessages    int64
	ActiveConversations int64
	FirstResponseAvgSec float64 // 0 se não houver dados
}

// TenantStats é o output completo do dashboard do tenant.
type TenantStats struct {
	Bloco1_Funil      FunilBlock
	Bloco2_WhatsApp   WhatsAppStats
	Bloco3_Responsive []ResponsiblePerformance
	Bloco4_TempoFunil []ColumnDwell
	Bloco5_Produtos   []ProductLeadsCount
	// Contexto de renderização:
	ScopeIsUser      bool   // true quando filtrado por responsible_user_id
	CurrentUserName  string // usado no título do Bloco 3 quando ScopeIsUser
	ActiveFunnelName string
}

type FunilBlock struct {
	StatusTotals  LeadStatusCount
	ColumnTotals  []ColumnLeadsCount
	ConversionPct float64 // won / (won+lost) * 100, 0 se ambos zero
	NewToday      int64
	NewThisWeek   int64
}
