// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type tagBaseModel struct {
	Id          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (model *tagBaseModel) updateState(t *client.TagRestApiGet) {
	model.Id = types.Int64Value(int64(t.Id))
	model.Name = types.StringValue(t.Name)
	if t.Description.IsNull() || t.Description.MustGet() == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(t.Description.MustGet())
	}
}
