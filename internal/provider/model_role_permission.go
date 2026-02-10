// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type rolePermissionBaseModel struct {
	RoleId      types.Int64  `tfsdk:"role_id"`
	RoleName    types.String `tfsdk:"role_name"`
	Permissions types.List   `tfsdk:"permissions"`
}

func (model *rolePermissionBaseModel) updateState(roleId int64, roleName string, permissions []client.SupersetRolePermissionApiGetList) {
	model.RoleId = types.Int64Value(roleId)
	model.RoleName = types.StringValue(roleName)
	model.Permissions = model.flattenPermissionsToList(permissions)
}

func (model *rolePermissionBaseModel) resolvePermissions(sourcePermissions []client.SupersetPermissionApiGetList) ([]client.SupersetRolePermissionApiGetList, []string) {
	var permissions []client.SupersetRolePermissionApiGetList
	if model.Permissions.IsNull() {
		return permissions, nil
	}

	sourcePermissionNameIdMap := make(map[string]int)
	for _, p := range sourcePermissions {
		sourcePermissionNameIdMap[p.Permission.Name+"_"+p.ViewMenu.Name] = p.Id
	}

	notFoundPermissions := make([]string, 0)

	permissionList := model.Permissions.Elements()
	for _, p := range permissionList {
		permissionObj, ok := p.(types.Object)
		if !ok || permissionObj.IsNull() {
			panic("unexpected type of permission attribute value")
		}

		permissionNameAttr, exists := permissionObj.Attributes()["permission_name"]
		if !exists {
			panic("permission_name attribute is missing in permission object")
		}
		_permissionNameAttrValue, ok := permissionNameAttr.(types.String)
		if !ok || _permissionNameAttrValue.IsNull() {
			panic("unexpected type of permission_name attribute value")
		}
		permissionNameAttrValue := _permissionNameAttrValue.ValueString()
		viewMenuNameAttr, exists := permissionObj.Attributes()["view_menu_name"]
		if !exists {
			panic("view_menu_name attribute is missing in permission object")
		}
		_viewMenuNameAttrValue, ok := viewMenuNameAttr.(types.String)
		if !ok || _viewMenuNameAttrValue.IsNull() {
			panic("unexpected type of view_menu_name attribute value")
		}

		viewMenuNameAttrValue := _viewMenuNameAttrValue.ValueString()
		fullPermissionName := permissionNameAttrValue + "_" + viewMenuNameAttrValue
		sourcePermissionId, exists := sourcePermissionNameIdMap[fullPermissionName]
		if !exists {
			notFoundPermissions = append(notFoundPermissions, fullPermissionName)
			continue
		}
		permissions = append(permissions, client.SupersetRolePermissionApiGetList{
			Id:             sourcePermissionId,
			PermissionName: permissionNameAttrValue,
			ViewMenuName:   viewMenuNameAttrValue,
		})
	}

	return permissions, notFoundPermissions
}

func (model *rolePermissionBaseModel) flattenPermissionsToList(permissions []client.SupersetRolePermissionApiGetList) types.List {
	permissionObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"permission_name": types.StringType,
			"view_menu_name":  types.StringType,
		},
	}

	elems := make([]attr.Value, 0, len(permissions))
	for _, p := range permissions {
		ov, _ := types.ObjectValue(
			permissionObjType.AttrTypes,
			map[string]attr.Value{
				"permission_name": types.StringValue(p.PermissionName),
				"view_menu_name":  types.StringValue(p.ViewMenuName),
			},
		)
		elems = append(elems, ov)
	}

	lv, _ := types.ListValue(permissionObjType, elems)
	return lv
}
