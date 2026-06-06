package playground

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	whatsappApp "github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// ContactLister lists contacts of a tenant. Adapter lives in Task 14.
type ContactLister interface {
	ListByTenant(ctx context.Context, tenantID string) ([]ContactSummary, error)
}

// ContactSummary is the view model the playground page consumes.
type ContactSummary struct {
	ID             string
	Name           string
	Phone          string
	WhatsAppID     string
	ConversationID string
}

// MessageLister pulls messages for a conversation in chronological order.
type MessageLister interface {
	ListByConversation(ctx context.Context, tenantID, conversationID string, limit int) ([]MessageView, error)
}

// MessageView is the view model for a rendered message.
type MessageView struct {
	ID        string
	Direction string // "incoming" | "outgoing"
	Content   string
	Timestamp time.Time
}

// ConversationResetter zeroes conversation state and moves the lead back to
// the entry column. Satisfied by *application.ResetConversationUseCase.
type ConversationResetter interface {
	Execute(ctx context.Context, tenantID, conversationID, source string) error
}

// MessageHistoryClearer deletes every message of a conversation, returning how
// many were removed. The playground reset uses it to wipe the history so the
// LLM starts from a true clean slate (no stale context to mimic).
type MessageHistoryClearer interface {
	ClearHistory(ctx context.Context, conversationID string) (int64, error)
}

// Handler exposes dev-only endpoints for simulating inbound messages and
// resetting conversation state.
type Handler struct {
	contacts ContactLister
	messages MessageLister
	receive  *whatsappApp.ReceiveMessageUseCase
	resetUC  ConversationResetter
	clearer  MessageHistoryClearer
	log      *zap.Logger
}

func NewHandler(
	contacts ContactLister,
	messages MessageLister,
	receive *whatsappApp.ReceiveMessageUseCase,
	resetUC ConversationResetter,
	clearer MessageHistoryClearer,
	log *zap.Logger,
) *Handler {
	return &Handler{
		contacts: contacts,
		messages: messages,
		receive:  receive,
		resetUC:  resetUC,
		clearer:  clearer,
		log:      log,
	}
}

// RenderPage renders the playground landing page with the tenant's contacts.
func (h *Handler) RenderPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	contacts, err := h.contacts.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("playground: list contacts failed", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "ai/playground.html", gin.H{"Error": "Erro ao carregar contatos"})
		return
	}
	c.HTML(http.StatusOK, "ai/playground.html", gin.H{
		"Contacts":  contacts,
		"ActiveNav": "ai_playground",
	})
}

// RenderConversation returns the fragment with messages for a contact's conversation.
func (h *Handler) RenderConversation(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	contactID := c.Param("contact_id")

	selected, err := h.findContact(c.Request.Context(), tenantID, contactID)
	if err != nil {
		h.log.Error("playground: list contacts failed", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	if selected == nil || selected.ConversationID == "" {
		c.Status(http.StatusNotFound)
		return
	}

	h.renderConversation(c, tenantID, selected)
}

// renderConversation lists the conversation's messages and renders the chat
// fragment. Shared by RenderConversation and HandleReset so a reset returns the
// (now empty) conversation immediately, updating #chat without a page refresh.
func (h *Handler) renderConversation(c *gin.Context, tenantID string, selected *ContactSummary) {
	msgs, err := h.messages.ListByConversation(c.Request.Context(), tenantID, selected.ConversationID, 100)
	if err != nil {
		h.log.Error("playground: list messages failed", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	c.HTML(http.StatusOK, "ai/playground_messages.html", gin.H{
		"Contact":        selected,
		"Messages":       msgs,
		"ConversationID": selected.ConversationID,
	})
}

// findContact is a small helper used by all handlers below.
func (h *Handler) findContact(ctx context.Context, tenantID, contactID string) (*ContactSummary, error) {
	list, err := h.contacts.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ID == contactID {
			return &list[i], nil
		}
	}
	return nil, nil
}

// HandleSendAsLead injects a message into the real inbound pipeline as if it
// came from the lead. Reuses ReceiveMessageUseCase so playground and WhatsApp
// exercise identical code paths.
func (h *Handler) HandleSendAsLead(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	contactID := c.Param("contact_id")
	content := c.PostForm("content")

	h.log.Debug("playground send",
		zap.String("tenant_id", tenantID),
		zap.String("contact_id", contactID),
		zap.String("content", content))

	if content == "" {
		h.log.Warn("playground send: empty content")
		c.Status(http.StatusBadRequest)
		return
	}

	selected, err := h.findContact(c.Request.Context(), tenantID, contactID)
	if err != nil {
		h.log.Error("playground send: find contact failed",
			zap.String("contact_id", contactID),
			zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	if selected == nil {
		h.log.Warn("playground send: contact not found",
			zap.String("contact_id", contactID))
		c.Status(http.StatusNotFound)
		return
	}

	event := whatsappDomain.IncomingMessage{
		TenantID:      tenantID,
		SenderJID:     selected.WhatsAppID,
		SenderName:    selected.Name,
		SenderPhone:   selected.Phone,
		Content:       content,
		WhatsAppMsgID: "playground-" + uuid.New().String(),
		Timestamp:     time.Now(),
	}
	if err := h.receive.Execute(c.Request.Context(), event); err != nil {
		h.log.Error("playground send: receive execute failed",
			zap.String("contact_id", contactID),
			zap.String("content", content),
			zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	// 204 — the client's HTMX polling picks up the new messages from the
	// /conversation endpoint.
	h.log.Debug("playground send: success",
		zap.String("contact_id", contactID))
	c.Status(http.StatusNoContent)
}

// HandleReset zeroes conversation state via ResetConversationUseCase.
func (h *Handler) HandleReset(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	contactID := c.Param("contact_id")

	selected, err := h.findContact(c.Request.Context(), tenantID, contactID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if selected == nil || selected.ConversationID == "" {
		c.Status(http.StatusNotFound)
		return
	}
	if err := h.resetUC.Execute(c.Request.Context(), tenantID, selected.ConversationID, "playground"); err != nil {
		h.log.Error("playground: reset failed", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	// Wipe the message history so the LLM starts from a clean slate. Done after
	// the reset so the just-sent confirmation is cleared too — otherwise the
	// reset phrase lingers in the context and the model parrots it back.
	deleted, err := h.clearer.ClearHistory(c.Request.Context(), selected.ConversationID)
	if err != nil {
		h.log.Error("playground: clear history failed",
			zap.String("conversation_id", selected.ConversationID),
			zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	h.log.Info("playground: history cleared",
		zap.String("conversation_id", selected.ConversationID),
		zap.Int64("deleted", deleted))
	// Re-render the now-empty conversation so HTMX swaps #chat immediately
	// (the reset form targets #chat); avoids needing a manual page refresh.
	h.renderConversation(c, tenantID, selected)
}
