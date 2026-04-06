package domain

var DefaultColumns = []struct {
	Name  string
	Type  ColumnType
	Color string
}{
	{"Novo", ColumnTypeEntry, "#22c55e"},
	{"Em Atendimento", ColumnTypeIntermediate, "#3b82f6"},
	{"Qualificado", ColumnTypeIntermediate, "#eab308"},
	{"Ganho", ColumnTypeWon, "#16a34a"},
	{"Perdido", ColumnTypeLost, "#ef4444"},
}
