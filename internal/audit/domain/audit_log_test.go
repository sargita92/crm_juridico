package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput() NewAuditLogInput {
	return NewAuditLogInput{
		ActorEmail: "admin@crm.com",
		Action:     ActionLoginSuccess,
		Entity:     "session",
		IP:         "192.168.0.1",
		UserAgent:  "Mozilla/5.0",
	}
}

// S2-C01 — campos obrigatorios preenchidos.
func TestNewAuditLog_HappyPath_FillsRequiredFields(t *testing.T) {
	in := validInput()
	log, err := NewAuditLog(in)

	require.NoError(t, err)
	require.NotNil(t, log)
	assert.NotEmpty(t, log.ID, "ID gerado automaticamente quando nao informado")
	assert.Equal(t, "admin@crm.com", log.ActorEmail)
	assert.Equal(t, ActionLoginSuccess, log.Action)
	assert.Equal(t, "session", log.Entity)
	assert.Equal(t, "192.168.0.1", log.IP)
	assert.Equal(t, "Mozilla/5.0", log.UserAgent)
	assert.False(t, log.CreatedAt.IsZero())
	// S2-C05 — created_at em UTC.
	assert.Equal(t, time.UTC, log.CreatedAt.Location())
	assert.Nil(t, log.TenantID)
	assert.Nil(t, log.UserID)
	assert.Nil(t, log.EntityID)
}

func TestNewAuditLog_PreservesProvidedID(t *testing.T) {
	in := validInput()
	in.ID = "fixed-id-1"
	log, err := NewAuditLog(in)

	require.NoError(t, err)
	assert.Equal(t, "fixed-id-1", log.ID)
}

func TestNewAuditLog_PreservesProvidedCreatedAt_NormalizedToUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	require.NoError(t, err)
	in := validInput()
	in.CreatedAt = time.Date(2026, 4, 24, 10, 0, 0, 0, loc)
	log, err := NewAuditLog(in)

	require.NoError(t, err)
	assert.Equal(t, time.UTC, log.CreatedAt.Location())
	assert.Equal(t, in.CreatedAt.UTC(), log.CreatedAt)
}

// S1-C20 — actor_email vazio rejeitado.
func TestNewAuditLog_RejectsEmptyActorEmail(t *testing.T) {
	in := validInput()
	in.ActorEmail = ""
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrActorEmailRequired)
}

func TestNewAuditLog_RejectsBlankActorEmail(t *testing.T) {
	in := validInput()
	in.ActorEmail = "   "
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrActorEmailRequired)
}

// S1-C19 / S2-C12 — Action invalida rejeitada.
func TestNewAuditLog_RejectsInvalidAction(t *testing.T) {
	in := validInput()
	in.Action = Action("hack.invalida")
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestNewAuditLog_RejectsEmptyAction(t *testing.T) {
	in := validInput()
	in.Action = Action("")
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrInvalidAction)
}

func TestNewAuditLog_RejectsEmptyEntity(t *testing.T) {
	in := validInput()
	in.Entity = ""
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrEntityRequired)
}

// S1-C21 — IP vazio rejeitado.
func TestNewAuditLog_RejectsEmptyIP(t *testing.T) {
	in := validInput()
	in.IP = ""
	_, err := NewAuditLog(in)
	assert.ErrorIs(t, err, ErrIPRequired)
}

// S2-C02 — TenantID nullable permitido.
func TestNewAuditLog_AllowsNullTenantID(t *testing.T) {
	in := validInput()
	in.TenantID = nil
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Nil(t, log.TenantID)
}

func TestNewAuditLog_PreservesTenantIDWhenProvided(t *testing.T) {
	in := validInput()
	id := "tenant-123"
	in.TenantID = &id
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	require.NotNil(t, log.TenantID)
	assert.Equal(t, "tenant-123", *log.TenantID)
}

// S2-C03 — UserID nullable + actor_email preenchido em falha de login.
func TestNewAuditLog_AllowsNullUserIDWithActorEmail(t *testing.T) {
	in := validInput()
	in.Action = ActionLoginFailure
	in.UserID = nil
	in.ActorEmail = "tentativa@externo.com"
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Nil(t, log.UserID)
	assert.Equal(t, "tentativa@externo.com", log.ActorEmail)
}

// S2-C07 — UserAgent truncado em 255 caracteres.
func TestNewAuditLog_TruncatesUserAgentTo255(t *testing.T) {
	in := validInput()
	in.UserAgent = strings.Repeat("a", 600)
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Len(t, log.UserAgent, MaxUserAgentLength)
	assert.Equal(t, strings.Repeat("a", MaxUserAgentLength), log.UserAgent)
}

func TestNewAuditLog_KeepsShortUserAgentUnchanged(t *testing.T) {
	in := validInput()
	in.UserAgent = "Mozilla/5.0 (compatible)"
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Equal(t, "Mozilla/5.0 (compatible)", log.UserAgent)
}

func TestNewAuditLog_AllowsEmptyUserAgent(t *testing.T) {
	in := validInput()
	in.UserAgent = ""
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Empty(t, log.UserAgent)
}

// S1-C22 / S2-C13 — sanitizacao de metadata com chaves proibidas (case-insensitive).
func TestNewAuditLog_SanitizesMetadataForbiddenKeys(t *testing.T) {
	in := validInput()
	in.Metadata = map[string]any{
		"reason":        "invalid_credentials",
		"password":      "p@ss",
		"password_hash": "$2a$...",
		"token":         "ey...",
		"secret":        "shh",
		"authorization": "Bearer xyz",
		"hash":          "abc",
		"api_key":       "sk-xxx",
		"cookie":        "session=...",
		"set-cookie":    "session=...",
	}
	log, err := NewAuditLog(in)

	require.NoError(t, err)
	require.NotNil(t, log.Metadata)
	assert.Equal(t, "invalid_credentials", log.Metadata["reason"])
	for _, forbidden := range []string{
		"password", "password_hash", "token", "secret",
		"authorization", "hash", "api_key", "cookie", "set-cookie",
	} {
		_, present := log.Metadata[forbidden]
		assert.False(t, present, "metadata deveria ter removido %q", forbidden)
	}
}

func TestNewAuditLog_SanitizesMetadataCaseInsensitive(t *testing.T) {
	in := validInput()
	in.Metadata = map[string]any{
		"PASSWORD":      "p",
		"Token":         "t",
		"AUTHORIZATION": "Bearer",
		"Secret":        "s",
		"Password_Hash": "h",
		"reason":        "ok",
	}
	log, err := NewAuditLog(in)

	require.NoError(t, err)
	require.Equal(t, 1, len(log.Metadata))
	assert.Equal(t, "ok", log.Metadata["reason"])
}

func TestNewAuditLog_NilMetadataStaysNil(t *testing.T) {
	in := validInput()
	in.Metadata = nil
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.Nil(t, log.Metadata)
}

func TestNewAuditLog_EmptyMetadataStaysEmpty(t *testing.T) {
	in := validInput()
	in.Metadata = map[string]any{}
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	assert.NotNil(t, log.Metadata)
	assert.Empty(t, log.Metadata)
}

func TestNewAuditLog_PreservesEntityID(t *testing.T) {
	in := validInput()
	id := "entity-42"
	in.EntityID = &id
	in.Entity = "tenant"
	log, err := NewAuditLog(in)
	require.NoError(t, err)
	require.NotNil(t, log.EntityID)
	assert.Equal(t, "entity-42", *log.EntityID)
}

// SanitizeMetadata e exposto para uso por casos de uso (defesa em profundidade
// no BuildDiff — Step 3). Confirma idempotencia.
func TestSanitizeMetadata_Idempotent(t *testing.T) {
	in := map[string]any{"reason": "x", "PASSWORD": "p"}
	first := SanitizeMetadata(in)
	second := SanitizeMetadata(first)
	assert.Equal(t, first, second)
}

func TestSanitizeMetadata_NilReturnsNil(t *testing.T) {
	assert.Nil(t, SanitizeMetadata(nil))
}
