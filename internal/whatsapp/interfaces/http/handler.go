package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// connectState tracks an ongoing connection attempt per tenant.
type connectState struct {
	lastQR string // most recent QR code received
	err    error  // error if connection failed
	done   bool   // connection process finished (connected or failed)
	mu     sync.Mutex
}

type Handler struct {
	sendMessageUC *application.SendMessageUseCase
	listConvsUC   *application.ListConversationsUseCase
	getMessagesUC *application.GetConversationMessagesUseCase
	connectUC     *application.ConnectWhatsAppUseCase
	statusUC      *application.GetConnectionStatusUseCase
	disconnectUC  *application.DisconnectWhatsAppUseCase
	unreadUC      *application.GetUnreadTotalUseCase
	log           *zap.Logger
	notesService  domain.LeadNotesService

	mu         sync.Mutex
	connStates map[string]*connectState
}

// SetNotesService late-binds the funnel-backed notes service. Wired in main.go after
// the funnel module is built (see SetLeadCreator for the same pattern).
func (h *Handler) SetNotesService(s domain.LeadNotesService) { h.notesService = s }

func NewHandler(
	sendMessageUC *application.SendMessageUseCase,
	listConvsUC *application.ListConversationsUseCase,
	getMessagesUC *application.GetConversationMessagesUseCase,
	connectUC *application.ConnectWhatsAppUseCase,
	statusUC *application.GetConnectionStatusUseCase,
	disconnectUC *application.DisconnectWhatsAppUseCase,
	unreadUC *application.GetUnreadTotalUseCase,
	log *zap.Logger,
) *Handler {
	return &Handler{
		sendMessageUC: sendMessageUC,
		listConvsUC:   listConvsUC,
		getMessagesUC: getMessagesUC,
		connectUC:     connectUC,
		statusUC:      statusUC,
		disconnectUC:  disconnectUC,
		unreadUC:      unreadUC,
		log:           log,
		connStates:    make(map[string]*connectState),
	}
}

// startConnect kicks off the connection in a goroutine if not already running.
func (h *Handler) startConnect(tenantID string) *connectState {
	h.mu.Lock()
	defer h.mu.Unlock()

	if st, ok := h.connStates[tenantID]; ok && !st.done {
		return st // already running
	}

	st := &connectState{}
	h.connStates[tenantID] = st

	go func() {
		h.log.Info("whatsapp connect goroutine started", zap.String("tenant_id", tenantID))

		qrChan, err := h.connectUC.Execute(context.Background(), tenantID)
		if err != nil {
			h.log.Error("whatsapp connect failed", zap.String("tenant_id", tenantID), zap.Error(err))
			st.mu.Lock()
			st.err = err
			st.done = true
			st.mu.Unlock()
			return
		}

		h.log.Info("whatsapp connect succeeded, waiting for QR codes", zap.String("tenant_id", tenantID))

		// Drain QR codes from the provider, keep the latest
		for qr := range qrChan {
			if qr == "already_connected" {
				h.log.Info("whatsapp already connected (session restored)", zap.String("tenant_id", tenantID))
				st.mu.Lock()
				st.done = true
				st.mu.Unlock()
				return
			}
			h.log.Info("whatsapp QR code received", zap.String("tenant_id", tenantID), zap.Int("qr_length", len(qr)))
			st.mu.Lock()
			st.lastQR = qr
			st.mu.Unlock()
		}

		h.log.Info("whatsapp QR channel closed", zap.String("tenant_id", tenantID))
		st.mu.Lock()
		st.done = true
		st.mu.Unlock()
	}()

	return st
}

func (h *Handler) RenderPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	status, _ := h.statusUC.Execute(c.Request.Context(), tenantID)
	connected := status != nil && status.Connected

	c.HTML(http.StatusOK, "whatsapp/page.html", gin.H{
		"Connected": connected,
		"TenantID":  tenantID,
		"ActiveNav": "whatsapp",
	})
}

func (h *Handler) HandleConnect(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	// Clean previous state if any
	h.mu.Lock()
	delete(h.connStates, tenantID)
	h.mu.Unlock()

	h.startConnect(tenantID)

	c.Header("HX-Redirect", "/tenant/whatsapp")
	c.Status(http.StatusOK)
}

// RenderQR returns the current QR state. Called by HTMX polling.
// Never blocks — reads from connectState populated by background goroutine.
func (h *Handler) RenderQR(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	// Already connected? Redirect.
	status, _ := h.statusUC.Execute(c.Request.Context(), tenantID)
	if status != nil && status.Connected {
		h.mu.Lock()
		delete(h.connStates, tenantID)
		h.mu.Unlock()

		c.Header("HX-Redirect", "/tenant/whatsapp")
		c.Status(http.StatusOK)
		return
	}

	// Ensure connection is running
	st := h.startConnect(tenantID)

	st.mu.Lock()
	qr := st.lastQR
	err := st.err
	done := st.done
	st.mu.Unlock()

	if err != nil {
		h.mu.Lock()
		delete(h.connStates, tenantID)
		h.mu.Unlock()

		c.HTML(http.StatusOK, "whatsapp/qr.html", gin.H{
			"Error": "Erro ao conectar. Tente novamente.",
		})
		return
	}

	if done && qr == "" {
		// Connected without QR (session restored)
		h.mu.Lock()
		delete(h.connStates, tenantID)
		h.mu.Unlock()

		c.Header("HX-Redirect", "/tenant/whatsapp")
		c.Status(http.StatusOK)
		return
	}

	if qr != "" {
		c.HTML(http.StatusOK, "whatsapp/qr.html", gin.H{
			"QRCode": qr,
		})
		return
	}

	// No QR yet, still connecting — show loading (HTMX will retry)
	c.HTML(http.StatusOK, "whatsapp/qr.html", gin.H{
		"Loading": true,
	})
}

// ServeQRImage generates a QR code PNG from the current QR data.
func (h *Handler) ServeQRImage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	h.mu.Lock()
	st, exists := h.connStates[tenantID]
	h.mu.Unlock()

	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	st.mu.Lock()
	qr := st.lastQR
	st.mu.Unlock()

	if qr == "" {
		c.Status(http.StatusNotFound)
		return
	}

	png, err := qrcode.Encode(qr, qrcode.Medium, 256)
	if err != nil {
		h.log.Error("failed to generate QR image", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}

func (h *Handler) RenderStatus(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	status, _ := h.statusUC.Execute(c.Request.Context(), tenantID)
	connected := status != nil && status.Connected

	// If just connected and was pairing, redirect to reload the page with conversations view
	if connected {
		h.mu.Lock()
		_, wasPairing := h.connStates[tenantID]
		delete(h.connStates, tenantID)
		h.mu.Unlock()

		if wasPairing {
			c.Header("HX-Redirect", "/tenant/whatsapp")
			c.Status(http.StatusOK)
			return
		}
	}

	c.HTML(http.StatusOK, "whatsapp/status.html", gin.H{
		"Connected": connected,
	})
}

// RenderUnreadBadge renderiza o pill do menu WhatsApp com a contagem de
// mensagens não lidas do tenant. Endpoint é curto de propósito — chamado toda
// vez que o SharedWorker propaga new-message/conversation-update (ver
// sse-worker.js) e a cada 30s como fallback.
func (h *Handler) RenderUnreadBadge(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	count, err := h.unreadUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("whatsapp unread badge query failed", zap.String("tenant_id", tenantID), zap.Error(err))
		count = 0
	}
	c.HTML(http.StatusOK, "whatsapp/unread_badge.html", gin.H{"Count": count})
}

func (h *Handler) RenderConversations(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

	output, err := h.listConvsUC.Execute(c.Request.Context(), application.ListConversationsInput{
		TenantID: tenantID,
		Search:   search,
		Page:     page,
		Limit:    20,
	})
	if err != nil {
		c.HTML(http.StatusInternalServerError, "whatsapp/conversations.html", gin.H{
			"Error": "Erro ao carregar conversas",
		})
		return
	}

	c.HTML(http.StatusOK, "whatsapp/conversations.html", gin.H{
		"Conversations": output.Conversations,
		"Total":         output.Total,
		"Page":          output.Page,
		"Search":        search,
	})
}

func (h *Handler) RenderChat(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")

	messages, err := h.getMessagesUC.Execute(c.Request.Context(), application.GetConversationMessagesInput{
		TenantID:       tenantID,
		ConversationID: convID,
		Limit:          50,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConversationNotFound) {
			c.HTML(http.StatusNotFound, "whatsapp/chat.html", gin.H{
				"Error": "Conversa nao encontrada",
			})
			return
		}
		c.HTML(http.StatusInternalServerError, "whatsapp/chat.html", gin.H{
			"Error": "Erro ao carregar mensagens",
		})
		return
	}

	c.HTML(http.StatusOK, "whatsapp/chat.html", gin.H{
		"ConversationID": convID,
		"Messages":       messages,
		"TenantID":       tenantID,
	})
}

func (h *Handler) RenderNewMessages(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")
	afterID := c.Query("after")

	messages, err := h.getMessagesUC.Execute(c.Request.Context(), application.GetConversationMessagesInput{
		TenantID:       tenantID,
		ConversationID: convID,
		AfterID:        afterID,
		Limit:          50,
	})
	if err != nil {
		c.Status(http.StatusNoContent)
		return
	}

	if len(messages) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	c.HTML(http.StatusOK, "whatsapp/messages_fragment.html", gin.H{
		"Messages": messages,
	})
}

func (h *Handler) HandleSendMessage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")
	content := c.PostForm("content")

	if content == "" {
		c.HTML(http.StatusBadRequest, "whatsapp/message.html", gin.H{
			"Error": "Mensagem nao pode ser vazia",
		})
		return
	}

	if len(content) > 4096 {
		c.HTML(http.StatusBadRequest, "whatsapp/message.html", gin.H{
			"Error": "Mensagem excede o tamanho maximo",
		})
		return
	}

	output, err := h.sendMessageUC.Execute(c.Request.Context(), application.SendMessageInput{
		TenantID:       tenantID,
		ConversationID: convID,
		Content:        content,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		errMsg := "Erro ao enviar mensagem"

		if errors.Is(err, domain.ErrConversationNotFound) {
			statusCode = http.StatusNotFound
			errMsg = "Conversa nao encontrada"
		} else if errors.Is(err, domain.ErrNotConnected) {
			errMsg = "WhatsApp desconectado"
		}

		if output != nil {
			c.HTML(statusCode, "whatsapp/message.html", gin.H{
				"Message": output,
				"Error":   errMsg,
			})
			return
		}

		c.HTML(statusCode, "whatsapp/message.html", gin.H{
			"Error": errMsg,
		})
		return
	}

	c.HTML(http.StatusOK, "whatsapp/message.html", gin.H{
		"Message": output,
	})
}

func (h *Handler) HandleDisconnect(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	_ = h.disconnectUC.Execute(c.Request.Context(), tenantID)

	h.mu.Lock()
	delete(h.connStates, tenantID)
	h.mu.Unlock()

	c.Header("HX-Redirect", "/tenant/whatsapp")
	c.Status(http.StatusOK)
}

// RenderNotesPanel renders the notes drawer body for the conversation. The note
// form is always present: a conversation without a lead yet gets one created on
// the first note (see WhatsAppNotesAdapter.AddNote).
func (h *Handler) RenderNotesPanel(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")

	notes, err := h.notesService.NotesForConversation(c.Request.Context(), tenantID, convID)
	if err != nil {
		h.log.Error("failed to load notes", zap.String("tenant_id", tenantID), zap.String("conversation_id", convID), zap.Error(err))
		c.HTML(http.StatusInternalServerError, "whatsapp/notes_panel.html", gin.H{
			"ConversationID": convID,
			"Error":          "Erro ao carregar notas! Caso persista entre em contato com o suporte",
		})
		return
	}

	c.HTML(http.StatusOK, "whatsapp/notes_panel.html", gin.H{
		"ConversationID": convID,
		"Notes":          notes,
	})
}

// HandleCreateNote creates a note on the current lead of the conversation and
// re-renders the notes panel with the updated list.
func (h *Handler) HandleCreateNote(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())
	convID := c.Param("id")
	content := c.PostForm("content")
	userID := c.GetString("user_id")

	if content == "" {
		c.HTML(http.StatusBadRequest, "whatsapp/notes_panel.html", gin.H{
			"ConversationID": convID,
			"Error":          "A nota nao pode ser vazia",
		})
		return
	}

	notes, err := h.notesService.AddNote(c.Request.Context(), tenantID, convID, content, userID)
	if err != nil {
		h.log.Error("failed to create note", zap.String("tenant_id", tenantID), zap.String("conversation_id", convID), zap.String("user_id", userID), zap.Error(err))
		c.HTML(http.StatusUnprocessableEntity, "whatsapp/notes_panel.html", gin.H{
			"ConversationID": convID,
			// A causa real (ex: tenant sem funil default, que impede criar o lead
			// onde a nota seria guardada) fica no log acima. Na tela vale a
			// orientacao: o operador nao resolve funil, mas sabe a quem recorrer
			// quando o erro nao e passageiro.
			"Error": "Erro ao adicionar nota! Caso persista entre em contato com o suporte",
		})
		return
	}

	c.HTML(http.StatusOK, "whatsapp/notes_panel.html", gin.H{
		"ConversationID": convID,
		"Notes":          notes,
	})
}
