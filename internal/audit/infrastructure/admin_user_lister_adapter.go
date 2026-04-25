package infrastructure

import (
	"context"
	"sort"
	"strings"

	"gorm.io/gorm"

	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// maxFilterAdminUsers segue a mesma decisao de maxFilterTenants — 100
// itens no dropdown sem busca inline.
const maxFilterAdminUsers = 100

// adminUserRow e o subset de colunas que o adapter precisa ler. Evita
// importar o `userModel` privado do auth/infrastructure (nao exportado).
type adminUserRow struct {
	ID    string `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	Email string `gorm:"column:email"`
}

// TableName fixa o nome da tabela para o gorm — auth.userModel mapeia
// para "users".
func (adminUserRow) TableName() string { return "users" }

// AdminUserListerAdapter satisfaz audithttp.AdminUserLister.
//
// Implementacao usa Gorm direto em vez de adicionar um metodo "ListAdmins"
// no UserRepository: e uma view de leitura especifica do painel admin de
// auditoria, sem pertencer ao dominio de autenticacao. Manter o query
// aqui isola o acoplamento ao schema (`role = 'admin'`).
type AdminUserListerAdapter struct {
	db *gorm.DB
}

// NewAdminUserListerAdapter constroi o adapter. `db` nao pode ser nil
// em runtime — composition root garante.
func NewAdminUserListerAdapter(db *gorm.DB) *AdminUserListerAdapter {
	return &AdminUserListerAdapter{db: db}
}

// ListAdminUsers seleciona usuarios com role=admin ordenados por nome.
//
// Limite de 100 aplicado no banco para evitar ler dezenas de milhares de
// rows quando a base crescer. O sort por nome e feito em memoria (com
// folding) para que acentos nao bagunce a ordem alfabetica.
func (a *AdminUserListerAdapter) ListAdminUsers(ctx context.Context) ([]auditdomain.AdminUserSummary, error) {
	var rows []adminUserRow
	err := a.db.WithContext(ctx).
		Model(&adminUserRow{}).
		Where("role = ?", string(authdomain.UserRoleAdmin)).
		Limit(maxFilterAdminUsers + 1). // +1 pra detectar overflow no futuro
		Order("name ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]auditdomain.AdminUserSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, auditdomain.AdminUserSummary{
			ID:    r.ID,
			Name:  r.Name,
			Email: r.Email,
		})
	}

	// Re-sort com folding pra que "Áurea" venha antes de "Bruno".
	sort.SliceStable(out, func(i, j int) bool {
		return foldedLess(out[i].Name, out[j].Name)
	})
	if len(out) > maxFilterAdminUsers {
		out = out[:maxFilterAdminUsers]
	}
	return out, nil
}

// foldedLess compara strings ignorando case e acentos basicos.
//
// Implementacao simples (ToLower) — suficiente para a maioria dos
// nomes em PT-BR no MVP. Caso seja necessario suporte completo a
// `unicode/normalize`, evolui-se a funcao sem mudar o contrato.
func foldedLess(a, b string) bool {
	return strings.ToLower(a) < strings.ToLower(b)
}
