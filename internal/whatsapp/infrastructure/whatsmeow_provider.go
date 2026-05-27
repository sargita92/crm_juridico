package infrastructure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type WhatsmeowProvider struct {
	mu       sync.RWMutex
	clients  map[string]*whatsmeow.Client
	stores   map[string]*sqlstore.Container
	handler  domain.IncomingMessageHandler
	log      *zap.Logger
	storeDir string
}

func NewWhatsmeowProvider(storeDir string, log *zap.Logger) *WhatsmeowProvider {
	return &WhatsmeowProvider{
		clients:  make(map[string]*whatsmeow.Client),
		stores:   make(map[string]*sqlstore.Container),
		log:      log,
		storeDir: storeDir,
	}
}

func (p *WhatsmeowProvider) Connect(ctx context.Context, tenantID string) (<-chan string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Disconnect existing client if any
	if existing, ok := p.clients[tenantID]; ok {
		existing.Disconnect()
		delete(p.clients, tenantID)
	}
	if existing, ok := p.stores[tenantID]; ok {
		_ = existing.Close()
		delete(p.stores, tenantID)
	}

	// Create SQLite store per tenant
	if err := os.MkdirAll(p.storeDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}
	dbPath := filepath.Join(p.storeDir, sanitizeTenantID(tenantID)+".db")
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath)
	container, err := sqlstore.New(ctx, "sqlite3", dsn, newWaLogger(p.log, "Database"))
	if err != nil {
		return nil, fmt.Errorf("failed to create whatsmeow store: %w", err)
	}
	if err := container.Upgrade(ctx); err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("failed to upgrade whatsmeow store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, newWaLogger(p.log, "Client"))
	p.stores[tenantID] = container

	// Output channel for our callers (simple strings)
	outChan := make(chan string, 10)

	// Already paired — just reconnect
	if client.Store.ID != nil {
		p.log.Info("whatsapp session found, reconnecting", zap.String("tenant_id", tenantID))

		// Register message handler
		client.AddEventHandler(func(evt interface{}) {
			p.handleEvent(tenantID, evt)
		})

		if err := client.Connect(); err != nil {
			_ = container.Close()
			return nil, fmt.Errorf("failed to reconnect: %w", err)
		}

		p.clients[tenantID] = client

		go func() {
			outChan <- "already_connected"
			close(outChan)
		}()

		return outChan, nil
	}

	// Not paired — use GetQRChannel (MUST be called before Connect)
	p.log.Info("whatsapp no session, starting QR pairing", zap.String("tenant_id", tenantID))

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Register message handler for after pairing
	client.AddEventHandler(func(evt interface{}) {
		p.handleEvent(tenantID, evt)
	})

	if err := client.Connect(); err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	p.clients[tenantID] = client

	// Goroutine drains whatsmeow QR channel and forwards codes to our simple string channel
	go func() {
		defer close(outChan)
		for item := range qrChan {
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				p.log.Info("whatsapp QR code received",
					zap.String("tenant_id", tenantID),
					zap.Int("code_length", len(item.Code)),
				)
				select {
				case outChan <- item.Code:
				default:
				}
			case "success":
				p.log.Info("whatsapp paired successfully", zap.String("tenant_id", tenantID))
				return
			case "timeout":
				p.log.Warn("whatsapp QR timeout", zap.String("tenant_id", tenantID))
				return
			default:
				if item.Error != nil {
					p.log.Error("whatsapp QR error",
						zap.String("tenant_id", tenantID),
						zap.String("event", item.Event),
						zap.Error(item.Error),
					)
				} else {
					p.log.Warn("whatsapp QR event",
						zap.String("tenant_id", tenantID),
						zap.String("event", item.Event),
					)
				}
				return
			}
		}
	}()

	return outChan, nil
}

func (p *WhatsmeowProvider) handleEvent(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		p.handleIncomingMessage(tenantID, v)
	case *events.Connected:
		connectionUp.WithLabelValues(tenantID).Set(1)
		p.log.Info("whatsapp connected", zap.String("tenant_id", tenantID))
	case *events.Disconnected:
		// Transient by nature: whatsmeow auto-reconnects (EnableAutoReconnect).
		// Logged so a flapping connection is visible, not a terminal failure.
		connectionUp.WithLabelValues(tenantID).Set(0)
		disconnectTotal.WithLabelValues(tenantID, "disconnected").Inc()
		p.log.Warn("whatsapp disconnected", zap.String("tenant_id", tenantID))
	case *events.StreamReplaced:
		// The same account connected elsewhere; whatsmeow stops reconnecting.
		connectionUp.WithLabelValues(tenantID).Set(0)
		disconnectTotal.WithLabelValues(tenantID, "stream_replaced").Inc()
		p.log.Warn("whatsapp stream replaced", zap.String("tenant_id", tenantID))
	case *events.StreamError:
		disconnectTotal.WithLabelValues(tenantID, "stream_error").Inc()
		p.log.Error("whatsapp stream error",
			zap.String("tenant_id", tenantID),
			zap.String("code", v.Code),
		)
	case *events.ConnectFailure:
		disconnectTotal.WithLabelValues(tenantID, "connect_failure").Inc()
		p.log.Error("whatsapp connect failure",
			zap.String("tenant_id", tenantID),
			zap.String("reason", v.Reason.String()),
			zap.String("message", v.Message),
		)
	case *events.TemporaryBan:
		disconnectTotal.WithLabelValues(tenantID, "temporary_ban").Inc()
		p.log.Error("whatsapp temporary ban",
			zap.String("tenant_id", tenantID),
			zap.Int("code", int(v.Code)),
			zap.Duration("expire", v.Expire),
		)
	case *events.KeepAliveTimeout:
		disconnectTotal.WithLabelValues(tenantID, "keepalive_timeout").Inc()
		p.log.Warn("whatsapp keepalive timeout", zap.String("tenant_id", tenantID))
	case *events.KeepAliveRestored:
		p.log.Info("whatsapp keepalive restored", zap.String("tenant_id", tenantID))
	case *events.LoggedOut:
		connectionUp.WithLabelValues(tenantID).Set(0)
		disconnectTotal.WithLabelValues(tenantID, "logged_out").Inc()
		p.log.Warn("whatsapp logged out",
			zap.String("tenant_id", tenantID),
			zap.Bool("on_connect", v.OnConnect),
		)
		p.mu.Lock()
		delete(p.clients, tenantID)
		p.mu.Unlock()
	}
}

func (p *WhatsmeowProvider) Disconnect(_ context.Context, tenantID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.clients[tenantID]; ok {
		client.Disconnect()
		delete(p.clients, tenantID)
	}
	if store, ok := p.stores[tenantID]; ok {
		_ = store.Close()
		delete(p.stores, tenantID)
	}
	return nil
}

func (p *WhatsmeowProvider) IsConnected(tenantID string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	client, ok := p.clients[tenantID]
	if !ok {
		return false
	}
	return client.IsConnected() && client.Store.ID != nil
}

func (p *WhatsmeowProvider) SendMessage(ctx context.Context, tenantID, recipientWhatsAppID, content string) (string, error) {
	p.mu.RLock()
	client, ok := p.clients[tenantID]
	p.mu.RUnlock()

	if !ok || !client.IsConnected() {
		return "", domain.ErrNotConnected
	}

	jid, err := types.ParseJID(recipientWhatsAppID)
	if err != nil {
		return "", fmt.Errorf("invalid recipient JID: %w", err)
	}

	resp, err := client.SendMessage(ctx, jid, &waE2E.Message{
		Conversation: proto.String(content),
	})
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return resp.ID, nil
}

func (p *WhatsmeowProvider) SetMessageHandler(handler domain.IncomingMessageHandler) {
	p.handler = handler
}

func (p *WhatsmeowProvider) Shutdown() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for tenantID, client := range p.clients {
		client.Disconnect()
		if store, ok := p.stores[tenantID]; ok {
			_ = store.Close()
		}
	}
	p.clients = make(map[string]*whatsmeow.Client)
	p.stores = make(map[string]*sqlstore.Container)
}

func (p *WhatsmeowProvider) handleIncomingMessage(tenantID string, msg *events.Message) {
	if p.handler == nil {
		return
	}
	if msg.Info.IsFromMe {
		return
	}
	if !isDirectMessage(msg.Info.Chat) {
		p.log.Debug("whatsmeow: skipping non-DM message",
			zap.String("tenant_id", tenantID),
			zap.String("chat_jid", msg.Info.Chat.String()),
			zap.String("chat_server", msg.Info.Chat.Server),
		)
		return
	}

	content := msg.Message.GetConversation()
	if content == "" {
		if ext := msg.Message.GetExtendedTextMessage(); ext != nil {
			content = ext.GetText()
		}
	}

	// Try to extract media. If present, download via whatsmeow. Caption becomes
	// the message content; text-only messages keep their existing behavior.
	media, mediaCaption := p.extractMedia(tenantID, msg)
	if media != nil && content == "" {
		content = mediaCaption
	}

	if content == "" && media == nil {
		return
	}

	senderJID := recipientJID(msg.Info.Sender)
	senderPhone := msg.Info.Sender.User

	p.handler(context.Background(), domain.IncomingMessage{
		TenantID:      tenantID,
		SenderJID:     senderJID,
		SenderName:    msg.Info.PushName,
		SenderPhone:   "+" + senderPhone,
		Content:       content,
		WhatsAppMsgID: msg.Info.ID,
		Timestamp:     msg.Info.Timestamp,
		Media:         media,
	})
}

// extractMedia inspects the incoming whatsmeow message for media parts
// (image/document/audio/video/sticker) and, if present, downloads the bytes
// using the tenant's whatsmeow client. It returns the payload (or nil) and
// the caption extracted from the media proto, if any.
//
// Download failures are logged and treated as "no media" — the caller may
// still fall back to persisting the text content (which is typically empty
// for media-only messages).
func (p *WhatsmeowProvider) extractMedia(tenantID string, msg *events.Message) (*domain.MediaPayload, string) {
	p.mu.RLock()
	client, ok := p.clients[tenantID]
	p.mu.RUnlock()
	if !ok || client == nil {
		return nil, ""
	}

	m := msg.Message
	if m == nil {
		return nil, ""
	}

	ctx := context.Background()

	switch {
	case m.ImageMessage != nil:
		img := m.ImageMessage
		bytes, err := client.Download(ctx, img)
		if err != nil {
			p.log.Error("whatsmeow: image download failed",
				zap.String("tenant_id", tenantID),
				zap.String("msg_id", msg.Info.ID),
				zap.Error(err))
			return nil, img.GetCaption()
		}
		name := fallbackMediaName(msg.Info.ID, img.GetMimetype(), "image")
		return &domain.MediaPayload{
			Type:     domain.MessageTypeImage,
			Name:     name,
			MimeType: img.GetMimetype(),
			Content:  bytes,
		}, img.GetCaption()

	case m.DocumentMessage != nil:
		doc := m.DocumentMessage
		bytes, err := client.Download(ctx, doc)
		if err != nil {
			p.log.Error("whatsmeow: document download failed",
				zap.String("tenant_id", tenantID),
				zap.String("msg_id", msg.Info.ID),
				zap.Error(err))
			return nil, doc.GetCaption()
		}
		name := doc.GetFileName()
		if name == "" {
			name = fallbackMediaName(msg.Info.ID, doc.GetMimetype(), "document")
		}
		return &domain.MediaPayload{
			Type:     domain.MessageTypeDocument,
			Name:     name,
			MimeType: doc.GetMimetype(),
			Content:  bytes,
		}, doc.GetCaption()

	case m.AudioMessage != nil:
		aud := m.AudioMessage
		bytes, err := client.Download(ctx, aud)
		if err != nil {
			p.log.Error("whatsmeow: audio download failed",
				zap.String("tenant_id", tenantID),
				zap.String("msg_id", msg.Info.ID),
				zap.Error(err))
			return nil, ""
		}
		return &domain.MediaPayload{
			Type:     domain.MessageTypeAudio,
			Name:     fallbackMediaName(msg.Info.ID, aud.GetMimetype(), "audio"),
			MimeType: aud.GetMimetype(),
			Content:  bytes,
		}, ""

	case m.VideoMessage != nil:
		vid := m.VideoMessage
		bytes, err := client.Download(ctx, vid)
		if err != nil {
			p.log.Error("whatsmeow: video download failed",
				zap.String("tenant_id", tenantID),
				zap.String("msg_id", msg.Info.ID),
				zap.Error(err))
			return nil, vid.GetCaption()
		}
		return &domain.MediaPayload{
			Type:     domain.MessageTypeVideo,
			Name:     fallbackMediaName(msg.Info.ID, vid.GetMimetype(), "video"),
			MimeType: vid.GetMimetype(),
			Content:  bytes,
		}, vid.GetCaption()

	case m.StickerMessage != nil:
		st := m.StickerMessage
		bytes, err := client.Download(ctx, st)
		if err != nil {
			p.log.Error("whatsmeow: sticker download failed",
				zap.String("tenant_id", tenantID),
				zap.String("msg_id", msg.Info.ID),
				zap.Error(err))
			return nil, ""
		}
		return &domain.MediaPayload{
			Type:     domain.MessageTypeSticker,
			Name:     fallbackMediaName(msg.Info.ID, st.GetMimetype(), "sticker"),
			MimeType: st.GetMimetype(),
			Content:  bytes,
		}, ""
	}

	return nil, ""
}

// fallbackMediaName builds a synthetic filename when whatsmeow does not
// provide one (images, audios, videos, stickers). Extension is inferred from
// MIME when possible; otherwise empty.
func fallbackMediaName(msgID, mime, kind string) string {
	ext := extFromMime(mime)
	if msgID == "" {
		return kind + ext
	}
	return kind + "-" + msgID + ext
}

func extFromMime(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav":
		return ".wav"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "application/pdf":
		return ".pdf"
	}
	return ""
}

func isDirectMessage(chat types.JID) bool {
	return chat.Server == types.DefaultUserServer
}

// recipientJID returns the user JID with no device part, suitable for use as
// a SendMessage recipient. Sender JIDs from incoming events include the device
// (e.g. "5511...:56@s.whatsapp.net"), which whatsmeow rejects when sending.
func recipientJID(sender types.JID) string {
	return sender.ToNonAD().String()
}

func sanitizeTenantID(id string) string {
	var b strings.Builder
	for _, c := range id {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

var _ domain.WhatsAppProvider = (*WhatsmeowProvider)(nil)
