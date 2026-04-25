package domain

import "time"

// Limites de paginacao alinhados ao wireframe (Tela 1) e a decisao do
// usuario 2026-04-24 (item 1): page_size invalido nao gera 400, e clampado.
const (
	// DefaultPageSize e usado quando o caller nao fornece page_size ou envia <= 0.
	DefaultPageSize = 10
	// MaxPageSize e o teto duro aplicado pelo Normalize.
	MaxPageSize = 100
)

// Filter agrupa os filtros e a paginacao da listagem de audit logs.
//
// E construido pelo handler HTTP (a partir do querystring) e passado para o
// caso de uso ListAuditLogs (Step 4) sem mais transformacoes — o repo
// recebe o Filter normalizado e usa Offset()/PageSize diretamente.
type Filter struct {
	TenantID *string
	UserID   *string
	Action   *Action
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

// Normalize aplica defaults e valida invariantes de filtro.
//
// Comportamento:
//   - PageSize <= 0           -> DefaultPageSize
//   - PageSize > MaxPageSize  -> clamp para MaxPageSize (decisao 1, 2026-04-24)
//   - Page <= 0               -> 1
//   - From > To (ambos non-nil) -> ErrInvalidPeriod
//   - Action nao nil e invalida -> ErrInvalidAction (decisao 2, 2026-04-24)
//
// Idempotente: chamar Normalize duas vezes produz o mesmo resultado.
func (f *Filter) Normalize() error {
	if f.PageSize <= 0 {
		f.PageSize = DefaultPageSize
	} else if f.PageSize > MaxPageSize {
		f.PageSize = MaxPageSize
	}
	if f.Page <= 0 {
		f.Page = 1
	}

	if f.Action != nil && !f.Action.IsValid() {
		return ErrInvalidAction
	}

	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		return ErrInvalidPeriod
	}

	return nil
}

// Offset retorna o deslocamento SQL (LIMIT/OFFSET) calculado a partir de
// Page e PageSize ja normalizados. Caller deve ter invocado Normalize antes;
// caso contrario o resultado para Page=0 sera negativo.
func (f Filter) Offset() int {
	return (f.Page - 1) * f.PageSize
}
