package playground

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	aiapp "github.com/sasrgita/crm-juridico/internal/ai/application"
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

// Handler exposes dev-only endpoints for simulating inbound messages and
// resetting conversation state.
type Handler struct {
	contacts ContactLister
	messages MessageLister
	receive  *whatsappApp.ReceiveMessageUseCase
	resetUC  *aiapp.ResetConversationUseCase
	log      *zap.Logger
}

func NewHandler(
	contacts ContactLister,
	messages MessageLister,
	receive *whatsappApp.ReceiveMessageUseCase,
	resetUC *aiapp.ResetConversationUseCase,
	log *zap.Logger,
) *Handler {
	return &Handler{
		contacts: contacts,
		messages: messages,
		receive:  receive,
		resetUC:  resetUC,
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

	if content == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	selected, err := h.findContact(c.Request.Context(), tenantID, contactID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if selected == nil {
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
		h.log.Error("playground: receive failed", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	// 204 — the client's HTMX polling picks up the new messages from the
	// /conversation endpoint.
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
	c.Status(http.StatusNoContent)
}
