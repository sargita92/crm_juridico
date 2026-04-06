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
		existing.Close()
		delete(p.stores, tenantID)
	}

	// Create SQLite store per tenant
	if err := os.MkdirAll(p.storeDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}
	dbPath := filepath.Join(p.storeDir, sanitizeTenantID(tenantID)+".db")
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath)
	container, err := sqlstore.New(ctx, "sqlite3", dsn, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create whatsmeow store: %w", err)
	}
	if err := container.Upgrade(ctx); err != nil {
		container.Close()
		return nil, fmt.Errorf("failed to upgrade whatsmeow store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		container.Close()
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	client := whatsmeow.NewClient(deviceStore, nil)
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
			container.Close()
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
		container.Close()
		return nil, fmt.Errorf("failed to get QR channel: %w", err)
	}

	// Register message handler for after pairing
	client.AddEventHandler(func(evt interface{}) {
		p.handleEvent(tenantID, evt)
	})

	if err := client.Connect(); err != nil {
		container.Close()
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
		p.log.Info("whatsapp connected", zap.String("tenant_id", tenantID))
	case *events.LoggedOut:
		p.log.Warn("whatsapp logged out", zap.String("tenant_id", tenantID))
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
		store.Close()
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
			store.Close()
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

	content := msg.Message.GetConversation()
	if content == "" {
		if ext := msg.Message.GetExtendedTextMessage(); ext != nil {
			content = ext.GetText()
		}
	}
	if content == "" {
		return
	}

	senderJID := msg.Info.Sender.String()
	senderPhone := msg.Info.Sender.User

	p.handler(context.Background(), domain.IncomingMessage{
		TenantID:      tenantID,
		SenderJID:     senderJID,
		SenderName:    msg.Info.PushName,
		SenderPhone:   "+" + senderPhone,
		Content:       content,
		WhatsAppMsgID: msg.Info.ID,
		Timestamp:     msg.Info.Timestamp,
	})
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
