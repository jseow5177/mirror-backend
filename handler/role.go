package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"github.com/rs/zerolog/log"
	"time"
)

type RoleHandler interface {
	GetActions(ctx context.Context, req *GetActionsRequest, res *GetActionsResponse) error
	CreateRole(ctx context.Context, req *CreateRoleRequest, res *CreateRoleResponse) error
	UpdateRoles(ctx context.Context, req *UpdateRolesRequest, res *UpdateRolesResponse) error
	GetRoles(ctx context.Context, req *GetRolesRequest, res *GetRolesResponse) error
}

type roleHandler struct {
	userRepo repo.UserRepo
	roleRepo repo.RoleRepo
}

func NewRoleHandler(userRepo repo.UserRepo, roleRepo repo.RoleRepo) RoleHandler {
	return &roleHandler{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

type GetRolesRequest struct {
	ContextInfo
}

type GetRolesResponse struct {
	Roles []*entity.Role `json:"roles,omitempty"`
}

var GetRolesValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *roleHandler) GetRoles(ctx context.Context, req *GetRolesRequest, res *GetRolesResponse) error {
	if err := GetRolesValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	roles, err := h.roleRepo.GetManyByTenantID(ctx, req.GetTenantID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get roles failed: %v", err)
		return err
	}

	res.Roles = roles

	return nil
}

type GetActionsRequest struct {
	ContextInfo
}

type GetActionsResponse struct {
	Actions map[string][]*entity.Action `json:"actions"`
}

var GetActions = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *roleHandler) GetActions(_ context.Context, req *GetActionsRequest, res *GetActionsResponse) error {
	if err := GetActions.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	res.Actions = entity.Actions

	return nil
}

type UpdateRoleRequest struct {
	ID       *uint64  `json:"id,omitempty"`
	Name     *string  `json:"name,omitempty"`
	RoleDesc *string  `json:"role_desc,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

func (req *UpdateRoleRequest) GetID() uint64 {
	if req != nil && req.ID != nil {
		return *req.ID
	}
	return 0
}

type UpdateRoleResponse struct {
	Role *entity.Role `json:"role,omitempty"`
}

func (req *UpdateRoleRequest) ToRole() *entity.Role {
	actions := make([]entity.ActionCode, 0)

	req.Actions = goutil.RemoveStrDuplicates(req.Actions)
	for _, action := range req.Actions {
		actions = append(actions, entity.ActionCode(action))
	}
	return &entity.Role{
		ID:         req.ID,
		Name:       req.Name,
		RoleDesc:   req.RoleDesc,
		Actions:    actions,
		UpdateTime: goutil.Uint64(uint64(time.Now().Unix())),
	}
}

var UpdateRoleValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"role_id":     &validator.UInt64{},
	"name":        ResourceNameValidator(true),
	"role_desc":   ResourceDescValidator(true),
	"actions": &validator.Slice{
		MinLen:    1,
		Validator: new(actionsValidator),
	},
})

type UpdateRolesRequest struct {
	ContextInfo
	Roles []*UpdateRoleRequest `json:"roles,omitempty"`
}

type UpdateRolesResponse struct {
	Role []*entity.Role `json:"roles,omitempty"`
}

var UpdateRolesValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"roles": &validator.Slice{
		MinLen:    1,
		Validator: UpdateRoleValidator,
	},
})

func (req *UpdateRolesRequest) ToRoles() []*entity.Role {
	roles := make([]*entity.Role, 0)
	for _, role := range req.Roles {
		roles = append(roles, role.ToRole())
	}
	return roles
}

func (h *roleHandler) UpdateRoles(ctx context.Context, req *UpdateRolesRequest, _ *UpdateRolesResponse) error {
	if err := UpdateRolesValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	var (
		roleUpdates = req.ToRoles()
		roleIDs     = make([]uint64, 0)
	)
	for _, role := range roleUpdates {
		roleIDs = append(roleIDs, role.GetID())
	}

	roles, err := h.roleRepo.GetManyByIDs(ctx, req.GetTenantID(), roleIDs)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get roles failed: %v", err)
		return err
	}

	updatedRoles := make([]*entity.Role, 0)
	for _, role := range roles {
		for _, roleUpdate := range roleUpdates {
			if role.GetID() == roleUpdate.GetID() {
				if role.Update(roleUpdate) {
					updatedRoles = append(updatedRoles, role)
				}
			}
		}
	}

	if len(updatedRoles) == 0 {
		log.Ctx(ctx).Info().Msg("no changes detected for roles")
		return nil
	}

	if err := h.roleRepo.UpdateMany(ctx, updatedRoles); err != nil {
		log.Ctx(ctx).Error().Msgf("update roles failed: %v", err)
		return err
	}

	return nil
}

type CreateRoleRequest struct {
	ContextInfo
	Name     *string  `json:"name,omitempty"`
	RoleDesc *string  `json:"role_desc,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

func (req *CreateRoleRequest) ToRole() *entity.Role {
	var (
		actions = make([]entity.ActionCode, 0)
		now     = uint64(time.Now().Unix())
	)

	req.Actions = goutil.RemoveStrDuplicates(req.Actions)
	for _, action := range req.Actions {
		actions = append(actions, entity.ActionCode(action))
	}
	return &entity.Role{
		Name:       req.Name,
		RoleDesc:   req.RoleDesc,
		Status:     entity.RoleStatusNormal,
		Actions:    actions,
		TenantID:   goutil.Uint64(req.GetTenantID()),
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}
}

type CreateRoleResponse struct {
	Role *entity.Role `json:"role,omitempty"`
}

var CreateRoleValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"name":        ResourceNameValidator(false),
	"role_desc":   ResourceDescValidator(false),
	"actions": &validator.Slice{
		MinLen:    1,
		Validator: new(actionsValidator),
	},
})

func (h *roleHandler) CreateRole(ctx context.Context, req *CreateRoleRequest, res *CreateRoleResponse) error {
	if err := CreateRoleValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	role := req.ToRole()

	id, err := h.roleRepo.Create(ctx, role)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create role failed: %v", err)
		return err
	}

	role.ID = goutil.Uint64(id)
	res.Role = role

	return nil
}
