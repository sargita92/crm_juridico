package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

var ErrCannotRemoveOwner = errors.New("cannot remove the tenant owner")

// UserOutput carries tenant-scoped user data.
type UserOutput struct {
	ID         string
	Name       string
	Email      string
	IsOwner    bool
	WhatsAppID string
}

// AdminUserOutput e o read model retornado pelos UCs de admin (F12 Step 7).
type AdminUserOutput struct {
	ID     string
	Name   string
	Email  string
	Role   domain.UserRole
	Status domain.UserStatus
}

// CreateAdminUserInput agrupa os parametros do CreateAdminUser UC.
//
// PasswordHash deve ser pre-computado pelo handler (mesmo padrao do
// invite_user). UC nao tem responsabilidade sobre hashing — mantem foco
// em persistencia + auditoria.
type CreateAdminUserInput struct {
	Name         string
	Email        string
	PasswordHash string
}

// UpdateAdminUserInput agrupa os campos editaveis em update.
//
// Decisao F12 Step 7: apenas Name e Email sao alteraveis via este UC.
// Mudanca de Status passa por DeactivateAdminUser/BlockAdminUser/
// UnblockAdminUser para preservar a granularidade do audit log
// (acoes distintas).
type UpdateAdminUserInput struct {
	ID    string
	Name  string
	Email string
}

// ManageUsersUseCase handles user management within a tenant and at the
// admin layer (F12 Step 7 expandiu o escopo para CRUD de usuarios admin
// com publicacao em audit log).
type ManageUsersUseCase struct {
	userRepo       domain.UserRepository
	userTenantRepo domain.UserTenantRepository
	publisher      auditapp.Publisher
}

// NewManageUsersUseCase creates a new ManageUsersUseCase.
//
// Publisher default e auditapp.NoopPublisher{} — testes antigos continuam
// passando sem precisar wirear o audit module.
func NewManageUsersUseCase(
	userRepo domain.UserRepository,
	userTenantRepo domain.UserTenantRepository,
) *ManageUsersUseCase {
	return &ManageUsersUseCase{
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		publisher:      auditapp.NoopPublisher{},
	}
}

// SetAuditPublisher injeta o publisher de auditoria (F12). Quando nil,
// mantem NoopPublisher default — UC permanece funcional.
//
// Invocado pelo composition root via auth.Module.SetAuditPublisher
// apos o audit module estar disponivel.
func (uc *ManageUsersUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

// ListTenantUsers returns all users belonging to the given tenant.
func (uc *ManageUsersUseCase) ListTenantUsers(ctx context.Context, tenantID string) ([]UserOutput, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.list_tenant_users",
		attribute.String("tenant.id", tenantID),
	)
	defer span.End()

	userTenants, err := uc.userTenantRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	out := make([]UserOutput, 0, len(userTenants))
	for _, ut := range userTenants {
		user, err := uc.userRepo.FindByID(ctx, ut.UserID)
		if err != nil {
			return nil, err
		}

		out = append(out, UserOutput{
			ID:         user.ID,
			Name:       user.Name,
			Email:      user.Email,
			IsOwner:    ut.IsOwner,
			WhatsAppID: ut.WhatsAppID,
		})
	}

	return out, nil
}

// RemoveFromTenant removes a user from the tenant. Returns an error if the user is the owner.
func (uc *ManageUsersUseCase) RemoveFromTenant(ctx context.Context, userID, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.remove_from_tenant",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	isOwner, err := uc.userTenantRepo.IsOwner(ctx, userID, tenantID)
	if err != nil {
		return err
	}
	if isOwner {
		return ErrCannotRemoveOwner
	}

	return uc.userTenantRepo.RemoveFromTenant(ctx, userID, tenantID)
}

// SetWhatsAppID updates the WhatsApp number linked to a user within a tenant.
func (uc *ManageUsersUseCase) SetWhatsAppID(ctx context.Context, userID, tenantID, whatsAppID string) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.set_whatsapp_id",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	return uc.userTenantRepo.UpdateWhatsAppID(ctx, userID, tenantID, whatsAppID)
}

// --- Admin User UCs (F12 Step 7) ---

// CreateAdminUser cria um usuario com Role=admin e publica audit log
// `user_admin.created`. Email duplicado retorna ErrUserEmailExists vindo
// do repo (sem audit log neste caso).
func (uc *ManageUsersUseCase) CreateAdminUser(ctx context.Context, in CreateAdminUserInput) (*AdminUserOutput, error) {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.create_admin_user",
		attribute.String("user.email", in.Email),
	)
	defer span.End()

	user, err := domain.NewUser(uuid.New().String(), in.Name, in.Email, in.PasswordHash, domain.UserRoleAdmin)
	if err != nil {
		return nil, err
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	uc.publishAdminUser(ctx, auditdomain.ActionUserAdminCreated, user, auditdomain.Metadata{
		"name":  user.Name,
		"email": user.Email,
	})

	return &AdminUserOutput{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status,
	}, nil
}

// UpdateAdminUser altera Name/Email de um usuario existente. Publica
// audit log `user_admin.updated` apenas quando Role=admin E houve
// mudanca real (BuildDiff != nil). Para alvo nao-admin, persiste mas
// NAO publica (escopo F12 Step 7 = so admin).
func (uc *ManageUsersUseCase) UpdateAdminUser(ctx context.Context, in UpdateAdminUserInput) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.update_admin_user",
		attribute.String("user.id", in.ID),
	)
	defer span.End()

	user, err := uc.userRepo.FindByID(ctx, in.ID)
	if err != nil {
		return err
	}

	before := snapshotAdminUser(user)

	user.Name = in.Name
	user.Email = in.Email

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if !user.IsAdmin() {
		// Alvo nao-admin: nao publica (escopo F12 Step 7).
		return nil
	}

	after := snapshotAdminUser(user)
	diff := auditapp.BuildDiff(before, after)
	if diff == nil {
		return nil
	}

	uc.publishAdminUser(ctx, auditdomain.ActionUserAdminUpdated, user, diff)
	return nil
}

// DeactivateAdminUser muda Status para inactive e publica
// `user_admin.deactivated` (apenas se Role=admin).
func (uc *ManageUsersUseCase) DeactivateAdminUser(ctx context.Context, userID string) error {
	return uc.changeAdminStatus(ctx, userID, domain.UserStatusInactive,
		auditdomain.ActionUserAdminDeactivated, "")
}

// BlockAdminUser bloqueia um usuario admin (Status=inactive + motivo no
// audit log). Publica `user_admin.blocked` com Metadata["motivo"].
//
// Decisao F12 Step 7: dominio nao tem status `blocked` separado de
// `inactive`; a distincao entre Deactivated e Blocked vive apenas no
// audit log (action diferente + motivo). UCs distintos preservam a
// semantica e o trail de auditoria, alinhado a S1-C10.
func (uc *ManageUsersUseCase) BlockAdminUser(ctx context.Context, userID, motivo string) error {
	return uc.changeAdminStatus(ctx, userID, domain.UserStatusInactive,
		auditdomain.ActionUserAdminBlocked, motivo)
}

// UnblockAdminUser reverte um bloqueio (Status=active) e publica
// `user_admin.unblocked` com Metadata["motivo"].
func (uc *ManageUsersUseCase) UnblockAdminUser(ctx context.Context, userID, motivo string) error {
	return uc.changeAdminStatus(ctx, userID, domain.UserStatusActive,
		auditdomain.ActionUserAdminUnblocked, motivo)
}

// changeAdminStatus e o helper compartilhado por Deactivate/Block/Unblock.
// Persiste a mudanca de status e publica a action correspondente quando
// o alvo e admin. Para alvo nao-admin, persiste sem publicar.
func (uc *ManageUsersUseCase) changeAdminStatus(
	ctx context.Context,
	userID string,
	newStatus domain.UserStatus,
	action auditdomain.Action,
	motivo string,
) error {
	ctx, span := observability.StartSpan(ctx, "auth.usecase.change_admin_status",
		attribute.String("user.id", userID),
		attribute.String("user.new_status", string(newStatus)),
	)
	defer span.End()

	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Status = newStatus
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if !user.IsAdmin() {
		return nil
	}

	meta := auditdomain.Metadata{}
	if motivo != "" {
		meta["motivo"] = motivo
	}

	uc.publishAdminUser(ctx, action, user, meta)
	return nil
}

// publishAdminUser e o helper unico que monta o RegisterAuditLogInput
// para acoes user_admin.* — TenantID sempre nil (admin global), Entity
// fixo, IP/UA do contexto via middleware. Publisher engole erros (F12 4.1).
func (uc *ManageUsersUseCase) publishAdminUser(
	ctx context.Context,
	action auditdomain.Action,
	user *domain.User,
	meta auditdomain.Metadata,
) {
	actorEmail, actorID := actorFromContext(ctx)
	id := user.ID
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   nil,
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     action,
		Entity:     "user_admin",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
		Metadata:   meta,
	})
}
