// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type userBaseModel struct {
	Id         types.Int64  `tfsdk:"id"`
	Username   types.String `tfsdk:"username"`
	Email      types.String `tfsdk:"email"`
	FirstName  types.String `tfsdk:"first_name"`
	LastName   types.String `tfsdk:"last_name"`
	Password   types.String `tfsdk:"password"`
    RoleNames  types.Set    `tfsdk:"role_names"`
	GroupNames types.Set    `tfsdk:"group_names"`
	Active     types.Bool   `tfsdk:"active"`
}

func (model *userBaseModel) resolveGroupIDsFromNames(sourceGroups []client.SupersetGroupApiGetList) ([]int, []string) {
	var ids []int
	if model.GroupNames.IsNull() {
		return ids, nil
	}

	sourceGroupNameIdMap := make(map[string]int)
	for _, g := range sourceGroups {
		sourceGroupNameIdMap[g.Name] = g.Id
	}

	notFoundGroups := make([]string, 0)

	groupNamesList := model.GroupNames.Elements()
	for _, g := range groupNamesList {
		nameAttrValue := g.(types.String).ValueString()
		sourceGroupId, exists := sourceGroupNameIdMap[nameAttrValue]
		if !exists {
			notFoundGroups = append(notFoundGroups, nameAttrValue)
			continue
		}
		ids = append(ids, sourceGroupId)
	}

	return ids, notFoundGroups
}

func (model *userBaseModel) resolveRoleIDsFromNames(sourceRoles []client.SupersetRoleApiGetList) ([]int, []string) {
    var ids []int
    if model.RoleNames.IsNull() {
        return ids, nil
    }

    sourceRoleNameIdMap := make(map[string]int)
    for _, r := range sourceRoles {
        sourceRoleNameIdMap[r.Name] = r.Id
    }

    notFoundRoles := make([]string, 0)

    roleNamesList := model.RoleNames.Elements()
    for _, r := range roleNamesList {
        nameAttrValue := r.(types.String).ValueString()
        sourceRoleId, exists := sourceRoleNameIdMap[nameAttrValue]
        if !exists {
            notFoundRoles = append(notFoundRoles, nameAttrValue)
            continue
        }
        ids = append(ids, sourceRoleId)
    }

    return ids, notFoundRoles
}

func (model *userBaseModel) updateState(u *client.SupersetUserApiGet, password *string) {
	model.Id = types.Int64Value(int64(u.Id))
	model.Username = types.StringValue(u.Username)
	model.Email = types.StringValue(u.Email)
	model.FirstName = types.StringValue(u.FirstName)
	model.LastName = types.StringValue(u.LastName)

	if u.Active.IsNull() {
		model.Active = types.BoolNull()
	} else {
		model.Active = types.BoolValue(u.Active.MustGet())
	}
    if password != nil {
        model.Password = types.StringValue(*password)
    } else {
        model.Password = types.StringNull()
    }
	model.RoleNames = model.flattenRoleNamesToSet(u)
	model.GroupNames = model.flattenGroupNamesToSet(u)
}

func (model *userBaseModel) flattenGroupNamesToSet(u *client.SupersetUserApiGet) types.Set {
	elems := make([]attr.Value, 0, len(u.Groups))
	for _, g := range u.Groups {
		elems = append(elems, types.StringValue(g.Name))
	}

	sv, _ := types.SetValue(types.StringType, elems)
	return sv
}

func (m *userBaseModel) flattenRoleNamesToSet(u *client.SupersetUserApiGet) types.Set {
    elems := make([]attr.Value, 0, len(u.Roles))
    for _, r := range u.Roles {
        elems = append(elems, types.StringValue(r.Name))
    }

    sv, _ := types.SetValue(types.StringType, elems)
    return sv
}
