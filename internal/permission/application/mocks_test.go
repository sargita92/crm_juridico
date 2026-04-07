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

func (m *mockPermissionRepo) FindByUserID(_ context.Context, userID string) ([]domain.Permission, error) {
	var result []domain.Permission
	for _, p := range m.perms {
		if p.UserID == userID {
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

func (m *mockPermissionRepo) DeleteByGroupAndResource(_ context.Context, groupID, resource string) error {
	var remaining []domain.Permission
	for _, p := range m.perms {
		if !(p.GroupID == groupID && p.Resource == resource) {
			remaining = append(remaining, p)
		}
	}
	m.perms = remaining
	return nil
}

func (m *mockPermissionRepo) DeleteByUserAndResource(_ context.Context, userID, resource string) error {
	var remaining []domain.Permission
	for _, p := range m.perms {
		if !(p.UserID == userID && p.Resource == resource) {
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
