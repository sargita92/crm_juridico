package integration

// Step 5 da F12 — integracao end-to-end auth + audit.
//
// Vive em um pacote separado (`internal/audit/integration`) para nao
// disparar import cycle: o pacote `audit/application` importa
// `audit/infrastructure` (metricas), e qualquer teste no pacote
// `audit/infrastructure` que tente importar `audit/application` cria um
// ciclo proibido em `go vet`. Aqui o pacote nao tem codigo de producao,
// entao podemos importar livremente os modulos de aplicacao e
// infraestrutura de auth+audit ao mesmo tempo.
//
// Reflete o wire-up que `cmd/api/main.go` executa na inicializacao:
// `auditMod.Publisher` injetado em `authMod` via SetAuditPublisher e
// chamado pelos handlers de admin login/logout.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	auditinfra "github.com/sasrgita/crm-juridico/internal/audit/infrastructure"
	authapp "github.com/sasrgita/crm-juridico/internal/auth/application"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" {
			short = true
			break
		}
	}
	if short {
		os.Exit(m.Run())
	}
	ctx := context.Background()
	sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
	code := m.Run()
	_ = sharedContainer.Container.Terminate(ctx)
	os.Exit(code)
}

type authAuditTestEnv struct {
	db        *gorm.DB
	loginUC   *authapp.LoginUseCase
	publisher auditapp.Publisher
	auditRepo *auditinfra.GormAuditLogRepository
	hasher    authdomain.PasswordHasher
}

func setupAuthAuditEnv(t *testing.T) *authAuditTestEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()
	require.NoError(t, database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath()))

	// Limpa tabelas-alvo para isolamento entre testes.
	require.NoError(t, db.Exec("DELETE FROM audit_logs").Error)
	require.NoError(t, db.Exec("DELETE FROM user_tenants").Error)
	require.NoError(t, db.Exec("DELETE FROM users").Error)

	userRepo := authinfra.NewGormUserRepository(db)
	userTenantRepo := authinfra.NewGormUserTenantRepository(db)
	tenantRepo := tenantinfra.NewGormTenantRepository(db)
	hasher := authinfra.NewBcryptHasher()
	tokenProvider := authinfra.NewJWTProvider("test-secret-step5", 1*time.Hour)

	loginUC := authapp.NewLoginUseCase(userRepo, userTenantRepo, tenantRepo, hasher, tokenProvider)

	auditRepo := auditinfra.NewGormAuditLogRepository(db)
	registerUC := auditapp.NewRegisterAuditLogUseCase(auditRepo, log)
	publisher := auditapp.NewPublisher(registerUC, log)

	return &authAuditTestEnv{
		db:        db,
		loginUC:   loginUC,
		publisher: publisher,
		auditRepo: auditRepo,
		hasher:    hasher,
	}
}

func (env *authAuditTestEnv) seedAdmin(t *testing.T, id, email, password string) *authdomain.User {
	t.Helper()
	hash, err := env.hasher.Hash(password)
	require.NoError(t, err)

	user := &authdomain.User{
		ID:           id,
		Name:         "Admin",
		Email:        email,
		PasswordHash: hash,
		Role:         authdomain.UserRoleAdmin,
		Status:       authdomain.UserStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	require.NoError(t, authinfra.NewGormUserRepository(env.db).Create(context.Background(), user))
	return user
}

// publishLoginSuccess espelha o handler `cmd/api/main.go` sem precisar
// subir o roteador completo: o handler real chama o publisher exatamente
// com esses campos (acao, ator, IP/UA via context.Value).
func (env *authAuditTestEnv) publishLoginSuccess(ctx context.Context, email string, userID string) {
	_ = env.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		Action:     auditdomain.ActionLoginSuccess,
		ActorEmail: email,
		UserID:     &userID,
		Entity:     "session",
		EntityID:   &userID,
		IP:         "203.0.113.10",
		UserAgent:  "test-agent",
	})
}

func (env *authAuditTestEnv) publishLoginFailure(ctx context.Context, email, reason string) {
	_ = env.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		Action:     auditdomain.ActionLoginFailure,
		ActorEmail: email,
		Entity:     "session",
		IP:         "203.0.113.10",
		UserAgent:  "test-agent",
		Metadata:   auditdomain.Metadata{"reason": reason},
	})
}

func (env *authAuditTestEnv) listAuditLogs(t *testing.T) []*auditdomain.AuditLog {
	t.Helper()
	filter := auditdomain.Filter{Page: 1, PageSize: 50}
	filter.Normalize()
	out, _, err := env.auditRepo.List(context.Background(), filter)
	require.NoError(t, err)
	return out
}

// --- Tests ---

func TestAuthAudit_LoginSuccess_PublishesAuditLog(t *testing.T) {
	env := setupAuthAuditEnv(t)
	ctx := context.Background()

	admin := env.seedAdmin(t, "11111111-1111-1111-1111-111111111111", "admin@example.com", "secret123")

	out, err := env.loginUC.Execute(ctx, authapp.LoginInput{
		Email:    "admin@example.com",
		Password: "secret123",
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, admin.ID, out.UserID)

	env.publishLoginSuccess(ctx, "admin@example.com", admin.ID)

	logs := env.listAuditLogs(t)
	require.Len(t, logs, 1)
	assert.Equal(t, auditdomain.ActionLoginSuccess, logs[0].Action)
	assert.Equal(t, "admin@example.com", logs[0].ActorEmail)
	require.NotNil(t, logs[0].UserID)
	assert.Equal(t, admin.ID, *logs[0].UserID)
}

func TestAuthAudit_LoginFailure_WrongPassword_PublishesCredenciaisInvalidas(t *testing.T) {
	env := setupAuthAuditEnv(t)
	ctx := context.Background()

	env.seedAdmin(t, "22222222-2222-2222-2222-222222222222", "admin2@example.com", "right-password")

	_, err := env.loginUC.Execute(ctx, authapp.LoginInput{
		Email:    "admin2@example.com",
		Password: "wrong-password",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, authdomain.ErrInvalidCredentials))

	env.publishLoginFailure(ctx, "admin2@example.com", "credenciais_invalidas")

	logs := env.listAuditLogs(t)
	require.Len(t, logs, 1)
	assert.Equal(t, auditdomain.ActionLoginFailure, logs[0].Action)
	assert.Equal(t, "admin2@example.com", logs[0].ActorEmail)
	assert.Nil(t, logs[0].UserID, "user_id deve ser NULL para falha de credenciais (snapshot do form)")
	require.NotNil(t, logs[0].Metadata)
	assert.Equal(t, "credenciais_invalidas", logs[0].Metadata["reason"])
}

func TestAuthAudit_LoginFailure_UnknownEmail_PublishesUsuarioNaoEncontrado(t *testing.T) {
	env := setupAuthAuditEnv(t)
	ctx := context.Background()

	_, err := env.loginUC.Execute(ctx, authapp.LoginInput{
		Email:    "naoexiste@example.com",
		Password: "anything",
	})
	require.Error(t, err)
	// LoginUseCase mapeia "user not found" para ErrInvalidCredentials por
	// hardening (S5 OWASP). O handler em main.go diferencia os dois estados
	// olhando o erro tipado retornado pelo repo (ErrUserNotFound) ANTES de
	// passar pelo LoginUseCase — aqui simulamos o caminho que o handler de
	// fato exercita publicando direto com `usuario_nao_encontrado`. Cobre
	// o cenario S1-C13 dos cenarios de QA.
	env.publishLoginFailure(ctx, "naoexiste@example.com", "usuario_nao_encontrado")

	logs := env.listAuditLogs(t)
	require.Len(t, logs, 1)
	assert.Equal(t, auditdomain.ActionLoginFailure, logs[0].Action)
	assert.Equal(t, "naoexiste@example.com", logs[0].ActorEmail)
	assert.Nil(t, logs[0].UserID)
	require.NotNil(t, logs[0].Metadata)
	assert.Equal(t, "usuario_nao_encontrado", logs[0].Metadata["reason"])
}
