package infrastructure

import (
	"context"
	"sort"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

// maxFilterTenants e o teto duro de itens enviados para o dropdown de
// filtro. Decisao: no MVP nao temos dropdown searchable; mostramos os
// primeiros 100 ordenados alfabeticamente. Tenants alem disso nao
// aparecem ate o admin escrever o ID manualmente no `?tenant_id=...`.
//
// TODO: MVP — dropdown searchable fica para evolucao (decisao
// 2026-04-24). Quando passar de 100 tenants, o template renderiza um
// option "(mostrando 100 mais recentes — refine pelo ID)" puramente
// visual.
const maxFilterTenants = 100

// TenantListerAdapter satisfaz audithttp.TenantLister consumindo o
// TenantRepository ja existente. Mantemos o adapter aqui (em
// `internal/audit/infrastructure`) em vez de em
// `internal/tenant/infrastructure` para que o pacote audit nao crie
// dependencia circular caso o tenant queira eventualmente importar
// algo do audit.
type TenantListerAdapter struct {
	repo tenantdomain.TenantRepository
}

// NewTenantListerAdapter constroi o adapter. `repo` nao pode ser nil em
// runtime; o composition root e quem garante isso.
func NewTenantListerAdapter(repo tenantdomain.TenantRepository) *TenantListerAdapter {
	return &TenantListerAdapter{repo: repo}
}

// ListTenants devolve os primeiros `maxFilterTenants` tenants ordenados
// por nome (case-insensitive). Em caso de erro do repo propaga para o
// caller — o handler trata como "dropdown vazio" para nao quebrar a
// pagina inteira por falha em um filtro.
func (a *TenantListerAdapter) ListTenants(ctx context.Context) ([]auditdomain.TenantSummary, error) {
	all, err := a.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]auditdomain.TenantSummary, 0, len(all))
	for _, t := range all {
		out = append(out, auditdomain.TenantSummary{ID: t.ID, Name: t.Name})
	}

	sort.SliceStable(out, func(i, j int) bool {
		return foldedLess(out[i].Name, out[j].Name)
	})
	if len(out) > maxFilterTenants {
		out = out[:maxFilterTenants]
	}
	return out, nil
}
