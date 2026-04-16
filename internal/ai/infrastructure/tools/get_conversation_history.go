package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sasrgita/crm-juridico/internal/ai/domain"
)

// HistoryMessage is a self-contained message type for conversation history,
// independent of any whatsapp domain types.
type HistoryMessage struct {
	Role      string // "user" | "assistant" | etc.
	Content   string
	Timestamp time.Time
}

// ConversationHistoryFetcher defines the contract for fetching conversation messages.
type ConversationHistoryFetcher interface {
	FindMessages(ctx context.Context, conversationID string, limit int) ([]HistoryMessage, error)
}

const defaultHistoryLimit = 20

// GetConversationHistoryTool retrieves recent messages from a conversation.
type GetConversationHistoryTool struct {
	fetcher ConversationHistoryFetcher
}

func NewGetConversationHistoryTool(fetcher ConversationHistoryFetcher) *GetConversationHistoryTool {
	return &GetConversationHistoryTool{fetcher: fetcher}
}

func (t *GetConversationHistoryTool) Definition() domain.ToolDefinition {
	def, _ := domain.NewToolDefinition(
		"get_conversation_history",
		"Retorna o historico de mensagens de uma conversa",
		domain.ToolCategoryDataQuery,
		map[string]domain.ParameterDef{
			"conversation_id": {
				Type:        "string",
				Description: "ID da conversa",
				Required:    true,
			},
			"limit": {
				Type:        "number",
				Description: "Numero maximo de mensagens (padrao: 20)",
				Required:    false,
			},
		},
	)
	return def
}

func (t *GetConversationHistoryTool) Execute(ctx context.Context, _ string, args map[string]interface{}) (*domain.ToolResult, error) {
	conversationID, _ := args["conversation_id"].(string)
	if conversationID == "" {
		r := domain.NewToolResult("", "parametro obrigatorio 'conversation_id' nao informado", true)
		return &r, nil
	}

	limit := defaultHistoryLimit
	if rawLimit, ok := args["limit"]; ok {
		switch v := rawLimit.(type) {
		case float64:
			if v > 0 {
				limit = int(v)
			}
		case int:
			if v > 0 {
				limit = v
			}
		}
	}

	messages, err := t.fetcher.FindMessages(ctx, conversationID, limit)
	if err != nil {
		r := domain.NewToolResult("", "erro ao buscar historico: "+err.Error(), true)
		return &r, nil
	}

	if len(messages) == 0 {
		r := domain.NewToolResult("", "nenhuma mensagem encontrada na conversa: "+conversationID, false)
		return &r, nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Historico da conversa %s (%d mensagens):\n", conversationID, len(messages)))
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04:05"),
			msg.Role,
			msg.Content,
		))
	}

	r := domain.NewToolResult("", sb.String(), false)
	return &r, nil
}
