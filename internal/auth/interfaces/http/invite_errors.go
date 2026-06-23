package http

import (
	"errors"
	"net/http"

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// inviteErrorMessage devolve uma frase em PT para erros do fluxo de convite,
// adequada a exibição direta na tela pública.
func inviteErrorMessage(err error) string {
	switch {
	case errors.Is(err, domain.ErrInviteTokenNotFound):
		return "Este convite não foi encontrado. Confirme o link com o administrador."
	case errors.Is(err, domain.ErrInviteTokenExpired):
		return "Este convite expirou. Solicite um novo link ao administrador."
	case errors.Is(err, domain.ErrInviteTokenUsed):
		return "Este convite já foi utilizado. Use seu e-mail e senha para entrar."
	case errors.Is(err, application.ErrUserAlreadyInTenant):
		return "Você já está associado a este escritório."
	default:
		return "Não foi possível processar o convite. Tente novamente em instantes."
	}
}

// inviteErrorStatus mapeia erros do convite ao status HTTP apropriado.
func inviteErrorStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrInviteTokenNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInviteTokenExpired), errors.Is(err, domain.ErrInviteTokenUsed):
		return http.StatusGone
	case errors.Is(err, application.ErrUserAlreadyInTenant):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// isInviteFatal indica se o erro torna o convite inutilizável (deve mostrar
// cartão de "convite indisponível") em vez de erro inline no formulário.
func isInviteFatal(err error) bool {
	return errors.Is(err, domain.ErrInviteTokenNotFound) ||
		errors.Is(err, domain.ErrInviteTokenExpired) ||
		errors.Is(err, domain.ErrInviteTokenUsed)
}
