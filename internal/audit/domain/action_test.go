package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAction_IsValid_AllConstants garante que cada constante exposta passa por IsValid.
// Cobre cenarios S1-C19 e S2-C12 do QA.
func TestAction_IsValid_AllConstants(t *testing.T) {
	valid := []Action{
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
	for _, a := range valid {
		t.Run(string(a), func(t *testing.T) {
			assert.True(t, a.IsValid(), "expected %q to be a valid action", a)
		})
	}
}

func TestAction_IsValid_RejectsArbitraryString(t *testing.T) {
	cases := []Action{
		"",
		"foo",
		"hack.invalida",
		"login.success ", // trailing space
		"LOGIN.SUCCESS",  // case sensitive
		"tenant.create",  // similar but not equal
	}
	for _, a := range cases {
		t.Run(string(a), func(t *testing.T) {
			assert.False(t, a.IsValid(), "expected %q to be invalid", a)
		})
	}
}

// TestAction_Humanized garante o texto em PT-BR para cada acao do enum.
// Apoia S4-C03 e S4-C05 (rotulos humanizados na UI).
func TestAction_Humanized_KnownActions(t *testing.T) {
	cases := map[Action]string{
		ActionLoginSuccess:         "Login bem-sucedido",
		ActionLoginFailure:         "Falha de login",
		ActionLogout:               "Logout",
		ActionTenantCreated:        "Escritório criado",
		ActionTenantUpdated:        "Escritório atualizado",
		ActionTenantDeactivated:    "Escritório desativado",
		ActionTenantBlocked:        "Escritório bloqueado",
		ActionTenantUnblocked:      "Escritório desbloqueado",
		ActionUserAdminCreated:     "Usuario admin criado",
		ActionUserAdminUpdated:     "Usuario admin atualizado",
		ActionUserAdminDeactivated: "Usuario admin desativado",
		ActionUserAdminBlocked:     "Usuario admin bloqueado",
		ActionUserAdminUnblocked:   "Usuario admin desbloqueado",
		ActionPermissionChanged:    "Permissao alterada",
	}
	for a, expected := range cases {
		t.Run(string(a), func(t *testing.T) {
			assert.Equal(t, expected, a.Humanized())
		})
	}
}

func TestAction_Humanized_FallbackForUnknown(t *testing.T) {
	a := Action("custom.unknown")
	// Para acoes fora do enum a UI exibe o codigo bruto como fallback.
	assert.Equal(t, "custom.unknown", a.Humanized())
}
