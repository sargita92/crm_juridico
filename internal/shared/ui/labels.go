// Package ui centraliza rótulos exibidos nos templates HTML.
// O backend persiste códigos em inglês (entry, view, leads, ...); a UI exibe em português.
package ui

var columnTypeLabels = map[string]string{
	"entry":        "Entrada",
	"intermediate": "Intermediária",
	"won":          "Ganho",
	"lost":         "Perdido",
}

// ColumnTypeLabel devolve o rótulo PT para o tipo de coluna do funil.
// Retorna o próprio valor de entrada se não houver tradução cadastrada.
func ColumnTypeLabel(t string) string {
	if label, ok := columnTypeLabels[t]; ok {
		return label
	}
	return t
}

var permissionActionLabels = map[string]string{
	"view":      "Visualizar",
	"read":      "Ler",
	"create":    "Criar",
	"update":    "Editar",
	"delete":    "Excluir",
	"manage":    "Gerenciar",
	"customize": "Personalizar",
}

// PermissionActionLabel devolve o rótulo PT para a ação de permissão.
func PermissionActionLabel(a string) string {
	if label, ok := permissionActionLabels[a]; ok {
		return label
	}
	return a
}

var permissionResourceLabels = map[string]string{
	"leads":       "Leads",
	"users":       "Usuários",
	"groups":      "Grupos",
	"funnels":     "Funis",
	"automations": "Automações",
	"products":    "Produtos",
	"specialists": "Especialistas",
	"files":       "Arquivos",
	"invites":     "Convites",
	"settings":    "Configurações",
}

// PermissionResourceLabel devolve o rótulo PT para o recurso de permissão.
func PermissionResourceLabel(r string) string {
	if label, ok := permissionResourceLabels[r]; ok {
		return label
	}
	return r
}
