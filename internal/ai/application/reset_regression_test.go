//go:build integration

package application_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// TestE2E_GreetingAfterReset_DoesNotReReset reproduces the reported playground
// bug: after a single "/reset", every subsequent plain message ("oi") was
// observed to re-trigger the reset confirmation ("Conversa reiniciada...").
//
// Correct behaviour: exactly ONE reset confirmation is ever sent — for the
// "/reset" message itself. The "oi" messages must flow through the normal
// (fake-provider) path and must NOT produce another reset confirmation.
func TestE2E_GreetingAfterReset_DoesNotReReset(t *testing.T) {
	env := setupTestEnv(t)
	defer env.Close()

	// 1. "/reset" creates the conversation and fires the single legitimate reset.
	env.SendInbound(t, fixtureTenantID, fixtureContactPhone, fixtureContactJID, "/reset")
	env.AwaitDebouncer(t, debounceWait)

	convID := env.MustConversationIDForPhone(t, fixtureTenantID, fixtureContactPhone)
	require.Equal(t, 1, countReiniciada(env.ListMessages(t, fixtureTenantID, convID)),
		"exactly one reset confirmation after the /reset itself")

	// 2. A plain greeting in a separate debounce window must NOT re-reset.
	env.SendInbound(t, fixtureTenantID, fixtureContactPhone, fixtureContactJID, "oi")
	env.AwaitDebouncer(t, debounceWait)
	require.Equal(t, 1, countReiniciada(env.ListMessages(t, fixtureTenantID, convID)),
		"first 'oi' after reset must not re-trigger reset")

	// 3. Another greeting in yet another window — still must NOT re-reset.
	env.SendInbound(t, fixtureTenantID, fixtureContactPhone, fixtureContactJID, "oi")
	env.AwaitDebouncer(t, debounceWait)
	require.Equal(t, 1, countReiniciada(env.ListMessages(t, fixtureTenantID, convID)),
		"second 'oi' after reset must not re-trigger reset")
}

func countReiniciada(msgs []whatsappdomain.Message) int {
	n := 0
	for _, m := range msgs {
		if strings.Contains(m.Content, "reiniciada") {
			n++
		}
	}
	return n
}
