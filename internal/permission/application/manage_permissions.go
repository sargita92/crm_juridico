package application

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	auditapp "github.com/sasrgita/crm-juridico/internal/audit/application"
	auditdomain "github.com/sasrgita/crm-juridico/internal/audit/domain"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// PermissionInput describes a resource+action pair to be assigned.
type PermissionInput struct {
	Resource string
	Action   string
}

// PermissionOutput is the read model returned after permission queries.
type PermissionOutput struct {
	ID       string
	Resource string
	Action   string
}

// ManagePermissionsUseCase handles group and user permission assignment.
type ManagePermissionsUseCase struct {
	permissions domain.PermissionRepository

	// userRepo e injetado via SetUserRepo (F12 Step 7) e e usado APENAS
	// para descobrir se o alvo de SetUserPermissions e admin antes de
	// publicar audit log. Quando nil, SetUserPermissions nao publica
	// (compativel com testes legados).
	userRepo authdomain.UserRepository

	// publisher e injetado via SetAuditPublisher (F12 Step 7). Quando nil
	// ou alvo nao-admin, nenhuma publicacao acontece.
	publisher auditapp.Publisher
}

func NewManagePermissionsUseCase(permissions domain.PermissionRepository) *ManagePermissionsUseCase {
	return &ManagePermissionsUseCase{
		permissions: permissions,
		publisher:   auditapp.NoopPublisher{},
	}
}

// SetUserRepo injeta o repositorio de usuarios. Necessario para que o
// SetUserPermissions saiba se deve publicar `permission.changed` (somente
// quando alvo for admin — escopo MVP F12). Quando nao injetado,
// SetUserPermissions opera sem audit log.
func (uc *ManagePermissionsUseCase) SetUserRepo(r authdomain.UserRepository) {
	uc.userRepo = r
}

// SetAuditPublisher injeta o publisher de auditoria. Quando nil, mantem
// NoopPublisher default — UC continua funcional sem audit em testes
// antigos. Erros do publisher sao engolidos pela politica F12.
func (uc *ManagePermissionsUseCase) SetAuditPublisher(p auditapp.Publisher) {
	if p == nil {
		uc.publisher = auditapp.NoopPublisher{}
		return
	}
	uc.publisher = p
}

// SetGroupPermissions replaces all permissions for the given group inside tenantID.
// It validates every input first, deletes existing permissions per resource, then
// creates the new set.
func (uc *ManagePermissionsUseCase) SetGroupPermissions(ctx context.Context, tenantID, groupID string, inputs []PermissionInput) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.set_group_permissions",
		attribute.String("tenant.id", tenantID),
		attribute.String("group.id", groupID),
	)
	defer span.End()

	// Validate all inputs before making any changes.
	for _, inp := range inputs {
		if _, err := domain.NewPermission(uuid.New().String(), tenantID, groupID, "", inp.Resource, inp.Action); err != nil {
			return err
		}
	}
	// Track which resources we've already cleared to avoid duplicate deletions.
	cleared := make(map[string]bool)
	for _, inp := range inputs {
		key := inp.Resource + "|" + inp.Action
		if !cleared[key] {
			if err := uc.permissions.DeleteByGroupAndResource(ctx, groupID, inp.Resource, inp.Action); err != nil {
				return err
			}
			cleared[key] = true
		}
	}
	for _, inp := range inputs {
		p, _ := domain.NewPermission(uuid.New().String(), tenantID, groupID, "", inp.Resource, inp.Action)
		if err := uc.permissions.Create(ctx, p); err != nil {
			return err
		}
	}
	infrastructure.ChangesTotal.WithLabelValues("group", "updated").Inc()
	return nil
}

// SetUserPermissions replaces all permissions for the given user inside tenantID.
//
// F12 Step 7: quando o alvo e usuario admin (Role=admin), publica
// `permission.changed` com diff das permissoes antes/depois. Para alvo
// nao-admin (ou quando userRepo nao foi injetado), opera sem audit log.
// TenantID = nil no log porque a decisao do usuario e que admin user e
// global.
func (uc *ManagePermissionsUseCase) SetUserPermissions(ctx context.Context, tenantID, userID string, inputs []PermissionInput) error {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.set_user_permissions",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	// Validate all inputs before making any changes.
	for _, inp := range inputs {
		if _, err := domain.NewPermission(uuid.New().String(), tenantID, "", userID, inp.Resource, inp.Action); err != nil {
			return err
		}
	}

	// Captura snapshot antes (para diff em audit log) — apenas se ha
	// chance de publicar. O FindByUserID e barato; rodamos sempre que
	// userRepo+publisher estao injetados (composicao normal em prod).
	var before []PermissionOutput
	shouldAttemptAudit := uc.userRepo != nil
	if shouldAttemptAudit {
		existing, err := uc.permissions.FindByUserID(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		before = toPermissionOutputs(existing)
	}

	cleared := make(map[string]bool)
	for _, inp := range inputs {
		key := inp.Resource + "|" + inp.Action
		if !cleared[key] {
			if err := uc.permissions.DeleteByUserAndResource(ctx, tenantID, userID, inp.Resource, inp.Action); err != nil {
				return err
			}
			cleared[key] = true
		}
	}
	for _, inp := range inputs {
		p, _ := domain.NewPermission(uuid.New().String(), tenantID, "", userID, inp.Resource, inp.Action)
		if err := uc.permissions.Create(ctx, p); err != nil {
			return err
		}
	}
	infrastructure.ChangesTotal.WithLabelValues("user", "updated").Inc()

	if shouldAttemptAudit {
		uc.publishUserPermissionChange(ctx, userID, before, inputs)
	}
	return nil
}

// publishUserPermissionChange resolve o user, valida que e admin (escopo
// MVP F12) e publica `permission.changed` com diff. Falha de userRepo
// nao aborta a operacao de negocio (auditoria e best-effort).
func (uc *ManagePermissionsUseCase) publishUserPermissionChange(
	ctx context.Context,
	userID string,
	before []PermissionOutput,
	afterInputs []PermissionInput,
) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		// Sem como decidir — politica F12: auditoria nao quebra fluxo.
		return
	}
	if !user.IsAdmin() {
		// Decisao do usuario (2026-04-24, item 5): MVP audita apenas
		// alteracoes em alvo admin. Tenant users ficam fora.
		return
	}

	// Monta diff usando uma chave unica "permissions" — alinha com a
	// orientacao do design ("diff dos campos que a operacao realmente
	// pode alterar"); este UC nao altera role/group/view_profile, so
	// pares resource:action.
	beforeMap := map[string]any{"permissions": permissionStrings(before)}
	afterMap := map[string]any{"permissions": permissionInputStrings(afterInputs)}

	diff := auditapp.BuildDiff(beforeMap, afterMap)
	if diff == nil {
		// Sem mudanca real — nao publica (consistente com Step 6).
		return
	}

	actorEmail, actorID := actorFromContext(ctx)
	id := user.ID
	_ = uc.publisher.Publish(ctx, auditapp.RegisterAuditLogInput{
		TenantID:   nil, // alvo admin global -> tenant_id NULL
		UserID:     actorID,
		ActorEmail: actorEmail,
		Action:     auditdomain.ActionPermissionChanged,
		Entity:     "user_admin",
		EntityID:   &id,
		IP:         middleware.IPFromContext(ctx),
		UserAgent:  middleware.UserAgentFromContext(ctx),
		Metadata:   diff,
	})
}

// permissionStrings normaliza []PermissionOutput em []string ordenado
// para que o diff seja estavel (ordem de FindByUserID nao e
// determinada). Ordenacao em-place via sort.Strings.
func permissionStrings(perms []PermissionOutput) []string {
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = p.Resource + ":" + p.Action
	}
	sort.Strings(out)
	return out
}

// permissionInputStrings espelha permissionStrings para o input.
func permissionInputStrings(perms []PermissionInput) []string {
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = p.Resource + ":" + p.Action
	}
	sort.Strings(out)
	return out
}

// GetGroupPermissions returns all permissions currently assigned to a group.
func (uc *ManagePermissionsUseCase) GetGroupPermissions(ctx context.Context, groupID string) ([]PermissionOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.get_group_permissions",
		attribute.String("group.id", groupID),
	)
	defer span.End()

	list, err := uc.permissions.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return toPermissionOutputs(list), nil
}

// GetUserPermissions returns all individual permissions assigned to a user
// within a tenant.
func (uc *ManagePermissionsUseCase) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]PermissionOutput, error) {
	ctx, span := observability.StartSpan(ctx, "permission.usecase.get_user_permissions",
		attribute.String("tenant.id", tenantID),
		attribute.String("user.id", userID),
	)
	defer span.End()

	list, err := uc.permissions.FindByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return toPermissionOutputs(list), nil
}

func toPermissionOutputs(perms []domain.Permission) []PermissionOutput {
	out := make([]PermissionOutput, len(perms))
	for i, p := range perms {
		out[i] = PermissionOutput{
			ID:       p.ID,
			Resource: p.Resource,
			Action:   p.Action,
		}
	}
	return out
}
