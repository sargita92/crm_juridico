package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaxUserAgentLength e o limite de caracteres do User-Agent persistido.
// Alinhado a 6.1 do design (`user_agent VARCHAR(255)`) e ao cenario S2-C07.
const MaxUserAgentLength = 255

// Metadata e o mapa livre de atributos por acao (motivo, diff, reason etc).
// Persistido como JSON na infraestrutura. O dominio so garante a sanitizacao
// (remocao de chaves sensiveis) — schema e validacao por acao sao do app.
type Metadata = map[string]any

// AuditLog e a entidade imutavel que representa um evento auditado.
//
// Imutabilidade: nao expomos setters nem `Update()`. Uma vez gravado, o log
// nao deve ser alterado (Story 2 criterio 6 do PO + decisoes do usuario).
type AuditLog struct {
	ID         string
	TenantID   *string
	UserID     *string
	ActorEmail string
	Action     Action
	Entity     string
	EntityID   *string
	IP         string
	UserAgent  string
	Metadata   Metadata
	CreatedAt  time.Time
}

// NewAuditLogInput agrupa os parametros do construtor para evitar uma
// assinatura com muitos argumentos posicionais. Campos opcionais (ID,
// CreatedAt, Metadata, ponteiros nullable) podem ser deixados zero.
type NewAuditLogInput struct {
	ID         string
	TenantID   *string
	UserID     *string
	ActorEmail string
	Action     Action
	Entity     string
	EntityID   *string
	IP         string
	UserAgent  string
	Metadata   Metadata
	CreatedAt  time.Time
}

// NewAuditLog valida os invariantes e retorna um AuditLog pronto para gravar.
//
// Invariantes:
//   - ActorEmail nao pode ser vazio (snapshot legivel mesmo apos delete do user).
//   - Action precisa pertencer ao enum (Action.IsValid()).
//   - Entity nao pode ser vazia (use "session" para acoes sem alvo de entidade).
//   - IP nao pode ser vazio.
//   - UserAgent e truncado em MaxUserAgentLength sem retornar erro.
//   - Metadata e sanitizada (chaves proibidas removidas, case-insensitive).
//   - ID e gerado via uuid.NewString() quando nao informado.
//   - CreatedAt e normalizado para UTC; quando zero, e setado para time.Now().UTC().
func NewAuditLog(in NewAuditLogInput) (*AuditLog, error) {
	if strings.TrimSpace(in.ActorEmail) == "" {
		return nil, ErrActorEmailRequired
	}
	if !in.Action.IsValid() {
		return nil, ErrInvalidAction
	}
	if in.Entity == "" {
		return nil, ErrEntityRequired
	}
	if in.IP == "" {
		return nil, ErrIPRequired
	}

	id := in.ID
	if id == "" {
		id = uuid.NewString()
	}

	createdAt := in.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	} else {
		createdAt = createdAt.UTC()
	}

	userAgent := in.UserAgent
	if len(userAgent) > MaxUserAgentLength {
		userAgent = userAgent[:MaxUserAgentLength]
	}

	metadata := SanitizeMetadata(in.Metadata)

	return &AuditLog{
		ID:         id,
		TenantID:   in.TenantID,
		UserID:     in.UserID,
		ActorEmail: in.ActorEmail,
		Action:     in.Action,
		Entity:     in.Entity,
		EntityID:   in.EntityID,
		IP:         in.IP,
		UserAgent:  userAgent,
		Metadata:   metadata,
		CreatedAt:  createdAt,
	}, nil
}

// forbiddenMetadataKeys lista as chaves que nunca podem ser persistidas em
// audit_logs.metadata. Comparacao e case-insensitive (ver SanitizeMetadata).
//
// Mantido em var nao-exportada propositalmente: callers nao devem mutar a
// lista; quem precisa estender vai pelo SanitizeMetadata.
var forbiddenMetadataKeys = map[string]struct{}{
	"password":      {},
	"password_hash": {},
	"token":         {},
	"secret":        {},
	"authorization": {},
	"hash":          {},
	"api_key":       {},
	"cookie":        {},
	"set-cookie":    {},
}

// SanitizeMetadata devolve uma copia de in sem as chaves proibidas.
//
// Comparacao e case-insensitive (PASSWORD, Password, password todas batem).
// Profundidade 1 — chaves aninhadas em mapas/structs nao sao inspecionadas
// (escopo do MVP). Quando in e nil, retorna nil para preservar o significado
// "metadata ausente" no banco.
func SanitizeMetadata(in Metadata) Metadata {
	if in == nil {
		return nil
	}
	out := make(Metadata, len(in))
	for k, v := range in {
		if IsForbiddenMetadataKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// IsForbiddenMetadataKey informa se uma chave esta na blocklist (case-insensitive).
//
// Exposto para reuso por helpers da camada de aplicacao (ex.: `BuildDiff`)
// que precisam aplicar a mesma politica de redaction sem duplicar a lista.
// Para sanitizar um Metadata inteiro, prefira `SanitizeMetadata`.
func IsForbiddenMetadataKey(key string) bool {
	_, banned := forbiddenMetadataKeys[strings.ToLower(key)]
	return banned
}
