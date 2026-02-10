// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type groupRoleBindingBaseModel struct {
	GroupId   types.Int64  `tfsdk:"group_id"`
	GroupName types.String `tfsdk:"group_name"`
	RoleNames types.Set    `tfsdk:"role_names"`
}

func (model *groupRoleBindingBaseModel) updateState(group *client.SupersetGroupApiGetList) {
	model.GroupId = types.Int64Value(int64(group.Id))
	model.GroupName = types.StringValue(group.Name)
	model.RoleNames = model.flattenRoleNamesToSet(group)
}

func (model *groupRoleBindingBaseModel) resolveRoleIds(sourceRoles []client.SupersetRoleApiGetList) ([]int, []string) {
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

	for _, p := range roleNamesList {
		pv, ok := p.(types.String)
		if !ok || pv.IsNull() {
			panic("unexpected type of role name attribute value")
		}
		roleNameAttrValue := pv.ValueString()
		sourceRoleId, exists := sourceRoleNameIdMap[roleNameAttrValue]
		if !exists {
			notFoundRoles = append(notFoundRoles, roleNameAttrValue)
			continue
		}
		ids = append(ids, sourceRoleId)
	}

	return ids, notFoundRoles
}

func (model *groupRoleBindingBaseModel) flattenRoleNamesToSet(group *client.SupersetGroupApiGetList) types.Set {
	roleNameType := types.StringType

	elems := make([]attr.Value, 0, len(group.Roles))
	for _, r := range group.Roles {
		elems = append(elems, types.StringValue(r.Name))
	}

	sv, _ := types.SetValue(roleNameType, elems)
	return sv
}
