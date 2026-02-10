// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type roleBaseModel struct {
	Id   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (model *roleBaseModel) updateState(r *client.SupersetRoleApiGet) {
	model.Id = types.Int64Value(int64(r.Id))
	model.Name = types.StringValue(r.Name)
}
