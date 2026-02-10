// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type groupBaseModel struct {
	Id    types.Int64  `tfsdk:"id"`
	Label types.String `tfsdk:"label"`
	Name  types.String `tfsdk:"name"`
}

func (model *groupBaseModel) updateState(g *client.SupersetGroupApiGet) {
	model.Id = types.Int64Value(int64(g.Id))
	if g.Label.IsNull() {
		model.Label = types.StringNull()
	} else {
		model.Label = types.StringValue(g.Label.MustGet())
	}
	model.Name = types.StringValue(g.Name)
}

//func (model *groupBaseModel) flattenUsersToList(g *client.SupersetGroupApiGet) types.List {
//
//	userObjType := types.ObjectType{
//		AttrTypes: map[string]attr.Type{
//			"id":       types.Int64Type,
//			"username": types.StringType,
//		},
//	}
//
//	elems := make([]attr.Value, 0, len(g.Users))
//	for _, u := range g.Users {
//		ov, _ := types.ObjectValue(
//			userObjType.AttrTypes,
//			map[string]attr.Value{
//				"id":       types.Int64Value(int64(u.Id)),
//				"username": types.StringValue(u.Username),
//			},
//		)
//		elems = append(elems, ov)
//	}
//
//	lv, _ := types.ListValue(userObjType, elems)
//	return lv
//}
//
//func (model *groupBaseModel) flattenRolesToList(g *client.SupersetGroupApiGet) types.List {
//
//	roleObjType := types.ObjectType{
//		AttrTypes: map[string]attr.Type{
//			"id":   types.Int64Type,
//			"name": types.StringType,
//		},
//	}
//
//	elems := make([]attr.Value, 0, len(g.Roles))
//	for _, r := range g.Roles {
//		ov, _ := types.ObjectValue(
//			roleObjType.AttrTypes,
//			map[string]attr.Value{
//				"id":   types.Int64Value(int64(r.Id)),
//				"name": types.StringValue(r.Name),
//			},
//		)
//		elems = append(elems, ov)
//	}
//
//	lv, _ := types.ListValue(roleObjType, elems)
//	return lv
//}
