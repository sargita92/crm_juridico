package domain

// Action e o enum tipado de acoes auditadas pelo F12.
//
// Nomenclatura segue o design (`<grupo>.<verbo no participio>`), que mapeia
// 1:1 com as colunas do banco e com o filtro `?action=` do dropdown da UI.
type Action string

const (
	// Auth / sessao
	ActionLoginSuccess Action = "login.success"
	ActionLoginFailure Action = "login.failure"
	ActionLogout       Action = "logout"

	// Tenants (admin global)
	ActionTenantCreated     Action = "tenant.created"
	ActionTenantUpdated     Action = "tenant.updated"
	ActionTenantDeactivated Action = "tenant.deactivated"
	ActionTenantBlocked     Action = "tenant.blocked"
	ActionTenantUnblocked   Action = "tenant.unblocked"

	// Usuarios admin (admin global)
	ActionUserAdminCreated     Action = "user_admin.created"
	ActionUserAdminUpdated     Action = "user_admin.updated"
	ActionUserAdminDeactivated Action = "user_admin.deactivated"
	ActionUserAdminBlocked     Action = "user_admin.blocked"
	ActionUserAdminUnblocked   Action = "user_admin.unblocked"

	// Permissao (alvo: usuario admin no MVP — decisao do usuario 2026-04-24, item 5)
	ActionPermissionChanged Action = "permission.changed"
)

// allActions mantem a whitelist canonica usada por IsValid e Humanized.
// E declarado como var (nao const) porque maps nao podem ser const em Go,
// mas e tratado como imutavel pelo pacote.
var humanizedActions = map[Action]string{
	ActionLoginSuccess:         "Login bem-sucedido",
	ActionLoginFailure:         "Falha de login",
	ActionLogout:               "Logout",
	ActionTenantCreated:        "Tenant criado",
	ActionTenantUpdated:        "Tenant atualizado",
	ActionTenantDeactivated:    "Tenant desativado",
	ActionTenantBlocked:        "Tenant bloqueado",
	ActionTenantUnblocked:      "Tenant desbloqueado",
	ActionUserAdminCreated:     "Usuario admin criado",
	ActionUserAdminUpdated:     "Usuario admin atualizado",
	ActionUserAdminDeactivated: "Usuario admin desativado",
	ActionUserAdminBlocked:     "Usuario admin bloqueado",
	ActionUserAdminUnblocked:   "Usuario admin desbloqueado",
	ActionPermissionChanged:    "Permissao alterada",
}

// IsValid retorna true se a Action pertence ao enum suportado.
//
// Comparacao e case-sensitive e nao faz trimming — qualquer normalizacao
// deve acontecer na borda HTTP antes de chegar aqui.
func (a Action) IsValid() bool {
	_, ok := humanizedActions[a]
	return ok
}

// Humanized retorna o rotulo em PT-BR usado pela UI (lista e detalhe).
//
// Para acoes fora do enum retorna o proprio codigo, garantindo que um log
// gravado por uma versao mais nova do app nunca quebre o rendering.
func (a Action) Humanized() string {
	if label, ok := humanizedActions[a]; ok {
		return label
	}
	return string(a)
}

// ActionOption e a representacao para alimentar o dropdown de filtros do
// painel admin (Tela 1 / S3-C05). Mantem o code (valor enviado no
// query string) e o label humanizado.
type ActionOption struct {
	Code  string
	Label string
}

// orderedActions garante que o dropdown apresente as acoes na ordem que
// faz mais sentido para o operador (auth, tenant, user_admin, permission)
// — diferente da iteracao de mapa, que e nao-deterministica.
var orderedActions = []Action{
	ActionLoginSuccess,
	ActionLoginFailure,
	ActionLogout,
	ActionTenantCreated,
	ActionTenantUpdated,
	ActionTenantDeactivated,
	ActionTenantBlocked,
	ActionTenantUnblocked,
	ActionUserAdminCreated,
	ActionUserAdminUpdated,
	ActionUserAdminDeactivated,
	ActionUserAdminBlocked,
	ActionUserAdminUnblocked,
	ActionPermissionChanged,
}

// AllActions devolve a lista de acoes na ordem canonica para popular o
// dropdown da UI. Cada entrada ja vem com o label humanizado, evitando
// que o template precise chamar funcMap por item.
func AllActions() []ActionOption {
	out := make([]ActionOption, 0, len(orderedActions))
	for _, a := range orderedActions {
		out = append(out, ActionOption{Code: string(a), Label: a.Humanized()})
	}
	return out
}
