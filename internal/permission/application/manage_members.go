package application

import (
	"context"

	"github.com/google/uuid"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/permission/domain"
)

// MemberOutput is the read model for a user-group membership.
type MemberOutput struct {
	ID      string
	UserID  string
	GroupID string
	Name    string
	Email   string
}

// ManageMembersUseCase handles adding, removing, and listing group members.
type ManageMembersUseCase struct {
	groups     domain.PermissionGroupRepository
	userGroups domain.UserGroupRepository
	userRepo   authdomain.UserRepository
}

func NewManageMembersUseCase(
	groups domain.PermissionGroupRepository,
	userGroups domain.UserGroupRepository,
	userRepo authdomain.UserRepository,
) *ManageMembersUseCase {
	return &ManageMembersUseCase{
		groups:     groups,
		userGroups: userGroups,
		userRepo:   userRepo,
	}
}

// AddMember adds a user to a group, verifying the group exists and the user is
// not already a member.
func (uc *ManageMembersUseCase) AddMember(ctx context.Context, userID, groupID, tenantID string) error {
	if _, err := uc.groups.FindByID(ctx, groupID); err != nil {
		return err
	}
	exists, err := uc.userGroups.Exists(ctx, userID, groupID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrUserAlreadyInGroup
	}
	ug, err := domain.NewUserGroup(uuid.New().String(), userID, groupID, tenantID)
	if err != nil {
		return err
	}
	return uc.userGroups.Create(ctx, ug)
}

// RemoveMember removes a user from a group.
func (uc *ManageMembersUseCase) RemoveMember(ctx context.Context, userID, groupID string) error {
	return uc.userGroups.Delete(ctx, userID, groupID)
}

// ListMembers returns all members of the given group with full user details.
func (uc *ManageMembersUseCase) ListMembers(ctx context.Context, groupID string) ([]MemberOutput, error) {
	list, err := uc.userGroups.FindByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberOutput, 0, len(list))
	for _, ug := range list {
		user, err := uc.userRepo.FindByID(ctx, ug.UserID)
		if err != nil {
			// Skip users that can't be found rather than failing the whole request
			continue
		}
		out = append(out, MemberOutput{
			ID:      user.ID,
			UserID:  ug.UserID,
			GroupID: ug.GroupID,
			Name:    user.Name,
			Email:   user.Email,
		})
	}
	return out, nil
}
