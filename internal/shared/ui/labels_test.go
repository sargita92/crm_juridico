package ui

import "testing"

func TestColumnTypeLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"entry", "Entrada"},
		{"intermediate", "Intermediária"},
		{"won", "Ganho"},
		{"lost", "Perdido"},
		{"", ""},
		{"desconhecido", "desconhecido"},
	}
	for _, c := range cases {
		if got := ColumnTypeLabel(c.in); got != c.want {
			t.Errorf("ColumnTypeLabel(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestPermissionActionLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"view", "Visualizar"},
		{"read", "Ler"},
		{"create", "Criar"},
		{"update", "Editar"},
		{"delete", "Excluir"},
		{"manage", "Gerenciar"},
		{"customize", "Personalizar"},
		{"unknown", "unknown"},
	}
	for _, c := range cases {
		if got := PermissionActionLabel(c.in); got != c.want {
			t.Errorf("PermissionActionLabel(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestPermissionResourceLabel(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"leads", "Leads"},
		{"users", "Usuários"},
		{"groups", "Grupos"},
		{"funnels", "Funis"},
		{"automations", "Automações"},
		{"products", "Produtos"},
		{"specialists", "Especialistas"},
		{"files", "Arquivos"},
		{"invites", "Convites"},
		{"settings", "Configurações"},
		{"outro", "outro"},
	}
	for _, c := range cases {
		if got := PermissionResourceLabel(c.in); got != c.want {
			t.Errorf("PermissionResourceLabel(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}
