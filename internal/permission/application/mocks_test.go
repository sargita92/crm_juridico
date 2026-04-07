package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// --- Helpers ---

func newTestPerm(tenantID, groupID, userID, resource, action string) domain.Permission {
	return domain.Permission{
		ID:        tenantID + "-" + groupID + "-" + userID + "-" + resource + "-" + action,
		TenantID:  tenantID,
		GroupID:   groupID,
		UserID:    userID,
		Resource:  resource,
		Action:    action,
		CreatedAt: time.Now(),
	}
}

func newTestUserGroup(userID, groupID, tenantID string) domain.UserGroup {
	return domain.UserGroup{
		ID:        userID + "-" + groupID,
		UserID:    userID,
		GroupID:   groupID,
		TenantID:  tenantID,
		CreatedAt: time.Now(),
	}
}

// --- Mock PermissionRepository ---

type mockPermissionRepo struct {
	perms []domain.Permission
}

func newMockPermissionRepo(perms ...domain.Permission) *mockPermissionRepo {
	return &mockPermissionRepo{perms: perms}
}

func (m *mockPermissionRepo) Create(_ context.Context, p *domain.Permission) error {
	m.perms = append(m.perms, *p)
	return nil
}

func (m *mockPermissionRepo) Delete(_ context.Context, id string) error {
	for i, p := range m.perms {
		if p.ID == id {
			m.perms = append(m.perms[:i], m.perms[i+1:]...)
			return nil
		}
	}
	return domain.ErrPermissionNotFound
}

func (m *mockPermissionRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.Permission, error) {
	var result []domain.Permission
	for _, p := range m.perms {
		if p.GroupID == groupID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermissionRepo) FindByUserID(_ context.Context, tenantID, userID string) ([]domain.Permission, error) {
	var result []domain.Permission
	for _, p := range m.perms {
		if p.TenantID == tenantID && p.UserID == userID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermissionRepo) FindByGroupIDs(_ context.Context, groupIDs []string) ([]domain.Permission, error) {
	if len(groupIDs) == 0 {
		return []domain.Permission{}, nil
	}
	idSet := make(map[string]bool, len(groupIDs))
	for _, id := range groupIDs {
		idSet[id] = true
	}
	var result []domain.Permission
	for _, p := range m.perms {
		if idSet[p.GroupID] {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPermissionRepo) DeleteByGroupAndResource(_ context.Context, groupID, resource, action string) error {
	var remaining []domain.Permission
	for _, p := range m.perms {
		if !(p.GroupID == groupID && p.Resource == resource && p.Action == action) {
			remaining = append(remaining, p)
		}
	}
	m.perms = remaining
	return nil
}

func (m *mockPermissionRepo) DeleteByUserAndResource(_ context.Context, tenantID, userID, resource, action string) error {
	var remaining []domain.Permission
	for _, p := range m.perms {
		if !(p.TenantID == tenantID && p.UserID == userID && p.Resource == resource && p.Action == action) {
			remaining = append(remaining, p)
		}
	}
	m.perms = remaining
	return nil
}

// --- Mock UserGroupRepository ---

type mockUserGroupRepo struct {
	userGroups []domain.UserGroup
}

func newMockUserGroupRepo(ugs ...domain.UserGroup) *mockUserGroupRepo {
	return &mockUserGroupRepo{userGroups: ugs}
}

func (m *mockUserGroupRepo) Create(_ context.Context, ug *domain.UserGroup) error {
	m.userGroups = append(m.userGroups, *ug)
	return nil
}

func (m *mockUserGroupRepo) Delete(_ context.Context, userID, groupID string) error {
	for i, ug := range m.userGroups {
		if ug.UserID == userID && ug.GroupID == groupID {
			m.userGroups = append(m.userGroups[:i], m.userGroups[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockUserGroupRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.UserGroup, error) {
	var result []domain.UserGroup
	for _, ug := range m.userGroups {
		if ug.GroupID == groupID {
			result = append(result, ug)
		}
	}
	return result, nil
}

func (m *mockUserGroupRepo) FindByUserAndTenant(_ context.Context, userID, tenantID string) ([]domain.UserGroup, error) {
	var result []domain.UserGroup
	for _, ug := range m.userGroups {
		if ug.UserID == userID && ug.TenantID == tenantID {
			result = append(result, ug)
		}
	}
	return result, nil
}

func (m *mockUserGroupRepo) Exists(_ context.Context, userID, groupID string) (bool, error) {
	for _, ug := range m.userGroups {
		if ug.UserID == userID && ug.GroupID == groupID {
			return true, nil
		}
	}
	return false, nil
}

// --- Mock PermissionGroupRepository ---

type mockGroupRepo struct {
	groups []domain.PermissionGroup
}

func newMockGroupRepo(groups ...domain.PermissionGroup) *mockGroupRepo {
	return &mockGroupRepo{groups: groups}
}

func (m *mockGroupRepo) Create(_ context.Context, g *domain.PermissionGroup) error {
	m.groups = append(m.groups, *g)
	return nil
}

func (m *mockGroupRepo) FindByID(_ context.Context, id string) (*domain.PermissionGroup, error) {
	for i, g := range m.groups {
		if g.ID == id {
			return &m.groups[i], nil
		}
	}
	return nil, domain.ErrGroupNotFound
}

func (m *mockGroupRepo) FindByIDAndTenantID(_ context.Context, id, tenantID string) (*domain.PermissionGroup, error) {
	for i, g := range m.groups {
		if g.ID == id && g.TenantID == tenantID {
			return &m.groups[i], nil
		}
	}
	return nil, domain.ErrGroupNotFound
}

func (m *mockGroupRepo) Update(_ context.Context, g *domain.PermissionGroup) error {
	for i, existing := range m.groups {
		if existing.ID == g.ID {
			m.groups[i] = *g
			return nil
		}
	}
	return domain.ErrGroupNotFound
}

func (m *mockGroupRepo) Delete(_ context.Context, id string) error {
	for i, g := range m.groups {
		if g.ID == id {
			m.groups = append(m.groups[:i], m.groups[i+1:]...)
			return nil
		}
	}
	return domain.ErrGroupNotFound
}

func (m *mockGroupRepo) FindByTenantID(_ context.Context, tenantID string) ([]domain.PermissionGroup, error) {
	var result []domain.PermissionGroup
	for _, g := range m.groups {
		if g.TenantID == tenantID {
			result = append(result, g)
		}
	}
	return result, nil
}

// --- Mock ViewProfileRepository ---

type mockViewProfileRepo struct {
	profiles []domain.ViewProfile
}

func newMockViewProfileRepo(profiles ...domain.ViewProfile) *mockViewProfileRepo {
	return &mockViewProfileRepo{profiles: profiles}
}

func (m *mockViewProfileRepo) CreateOrUpdate(_ context.Context, vp *domain.ViewProfile) error {
	for i, existing := range m.profiles {
		if existing.GroupID == vp.GroupID && existing.FunnelID == vp.FunnelID {
			m.profiles[i] = *vp
			return nil
		}
	}
	m.profiles = append(m.profiles, *vp)
	return nil
}

func (m *mockViewProfileRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.ViewProfile, error) {
	var result []domain.ViewProfile
	for _, vp := range m.profiles {
		if vp.GroupID == groupID {
			result = append(result, vp)
		}
	}
	return result, nil
}

func (m *mockViewProfileRepo) FindByGroupAndFunnel(_ context.Context, groupID, funnelID string) (*domain.ViewProfile, error) {
	for i, vp := range m.profiles {
		if vp.GroupID == groupID && vp.FunnelID == funnelID {
			return &m.profiles[i], nil
		}
	}
	return nil, domain.ErrViewProfileNotFound
}

func (m *mockViewProfileRepo) FindByGroupIDs(_ context.Context, groupIDs []string, funnelID string) ([]domain.ViewProfile, error) {
	idSet := make(map[string]bool, len(groupIDs))
	for _, id := range groupIDs {
		idSet[id] = true
	}
	var result []domain.ViewProfile
	for _, vp := range m.profiles {
		if idSet[vp.GroupID] && vp.FunnelID == funnelID {
			result = append(result, vp)
		}
	}
	return result, nil
}

func (m *mockViewProfileRepo) Delete(_ context.Context, groupID, funnelID string) error {
	for i, vp := range m.profiles {
		if vp.GroupID == groupID && vp.FunnelID == funnelID {
			m.profiles = append(m.profiles[:i], m.profiles[i+1:]...)
			return nil
		}
	}
	return domain.ErrViewProfileNotFound
}

// --- Mock GroupFunnelRepository ---

type mockGroupFunnelRepo struct {
	funnels []domain.GroupFunnel
}

func newMockGroupFunnelRepo(funnels ...domain.GroupFunnel) *mockGroupFunnelRepo {
	return &mockGroupFunnelRepo{funnels: funnels}
}

func (m *mockGroupFunnelRepo) CreateOrUpdate(_ context.Context, gf *domain.GroupFunnel) error {
	for i, existing := range m.funnels {
		if existing.GroupID == gf.GroupID && existing.FunnelID == gf.FunnelID {
			m.funnels[i] = *gf
			return nil
		}
	}
	m.funnels = append(m.funnels, *gf)
	return nil
}

func (m *mockGroupFunnelRepo) FindByGroupID(_ context.Context, groupID string) ([]domain.GroupFunnel, error) {
	var result []domain.GroupFunnel
	for _, gf := range m.funnels {
		if gf.GroupID == groupID {
			result = append(result, gf)
		}
	}
	return result, nil
}

func (m *mockGroupFunnelRepo) FindByFunnelID(_ context.Context, funnelID string) ([]domain.GroupFunnel, error) {
	var result []domain.GroupFunnel
	for _, gf := range m.funnels {
		if gf.FunnelID == funnelID {
			result = append(result, gf)
		}
	}
	return result, nil
}

func (m *mockGroupFunnelRepo) FindByFunnelAndColumn(_ context.Context, funnelID, columnID string) ([]domain.GroupFunnel, error) {
	var result []domain.GroupFunnel
	for _, gf := range m.funnels {
		if gf.FunnelID == funnelID && gf.CoversColumn(columnID) {
			result = append(result, gf)
		}
	}
	return result, nil
}

func (m *mockGroupFunnelRepo) Delete(_ context.Context, groupID, funnelID string) error {
	for i, gf := range m.funnels {
		if gf.GroupID == groupID && gf.FunnelID == funnelID {
			m.funnels = append(m.funnels[:i], m.funnels[i+1:]...)
			return nil
		}
	}
	return domain.ErrGroupFunnelNotFound
}

// --- Mock OwnerChecker ---

type mockOwnerChecker struct {
	owners map[string]bool // key: userID+"|"+tenantID
}

func newMockOwnerChecker() *mockOwnerChecker {
	return &mockOwnerChecker{owners: make(map[string]bool)}
}

func (m *mockOwnerChecker) setOwner(userID, tenantID string) {
	m.owners[userID+"|"+tenantID] = true
}

func (m *mockOwnerChecker) IsOwner(_ context.Context, userID, tenantID string) (bool, error) {
	return m.owners[userID+"|"+tenantID], nil
}

// --- Mock AdminChecker ---

type mockAdminChecker struct {
	admins map[string]bool
}

func newMockAdminChecker() *mockAdminChecker {
	return &mockAdminChecker{admins: make(map[string]bool)}
}

func (m *mockAdminChecker) setAdmin(userID string) {
	m.admins[userID] = true
}

func (m *mockAdminChecker) IsAdmin(_ context.Context, userID string) (bool, error) {
	return m.admins[userID], nil
}
