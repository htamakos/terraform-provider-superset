// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

var _ resource.Resource = &GroupRoleBindingResource{}
var _ resource.ResourceWithImportState = &GroupRoleBindingResource{}

func NewGroupRoleBindingResource() resource.Resource {
	return &GroupRoleBindingResource{}
}

type GroupRoleBindingResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type groupRoleBindingResourceModel struct {
	groupRoleBindingBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *GroupRoleBindingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_role_binding"
}

func (r *GroupRoleBindingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Resource for managing group role bindings in Superset.",

		Attributes: map[string]schema.Attribute{
			"group_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the group.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"group_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_names": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Role names to assign to the group.",
                Validators: []validator.Set{
                    setvalidator.SizeAtLeast(1),
                },
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, Update: true, Delete: true,
			}),
		},
	}
}

func (r *GroupRoleBindingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWrapper)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWrapper, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = c
}

func (r *GroupRoleBindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data groupRoleBindingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	sourceRoles, err := r.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}
	roleIds, notFoundRoles := data.resolveRoleIds(sourceRoles)
	if len(notFoundRoles) > 0 {
		resp.Diagnostics.AddError("Invalid Roles", fmt.Sprintf("The following roles were not found: %v", notFoundRoles))
		return
	}

	group, err := r.client.FindGroup(ctx, data.GroupName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find group with name %s: %s", data.GroupName.ValueString(), err))
		return
	}

    groupId := int(group.Id)
	err = r.client.AssignRolesToGroup(ctx, groupId, roleIds)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to assign roles to group ID %d: %s", groupId, err))
		return
	}

    group, err = r.client.FindGroup(ctx, data.GroupName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find group with name %s: %s", data.GroupName.ValueString(), err))
        return
    }

	data.updateState(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupRoleBindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data groupRoleBindingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	group, err := r.client.FindGroup(ctx, data.GroupName.ValueString())
	if client.IsNotFound(err) {
        resp.State.RemoveResource(ctx)
        return
    } else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get group with ID %d Name: %s: %s,", data.GroupId.ValueInt64(), data.GroupName.ValueString(), err))
		return
	}

	data.updateState(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GroupRoleBindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state groupRoleBindingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    sourceRoles, err := r.client.ListRoles(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
        return
    }
    roleIds, notFoundRoles := plan.resolveRoleIds(sourceRoles)
    if len(notFoundRoles) > 0 {
        resp.Diagnostics.AddError("Invalid Roles", fmt.Sprintf("The following roles were not found: %v", notFoundRoles))
        return
    }
    groupId := int(plan.GroupId.ValueInt64())
    err = r.client.AssignRolesToGroup(ctx, groupId, roleIds)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to assign roles to group ID %d: %s", groupId, err))
        return
    }
    group, err := r.client.FindGroup(ctx, plan.GroupName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find group with name %s: %s", plan.GroupName.ValueString(), err))
        return
    }

    state.updateState(group)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupRoleBindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupRoleBindingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    groupId := int(state.GroupId.ValueInt64())
    err := r.client.AssignRolesToGroup(ctx, groupId, []int{})
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove all roles from group ID %d: %s", groupId, err))
        return
    }

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *GroupRoleBindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Starting ImportState method", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.State.SetAttribute(ctx, path.Root("group_name"), req.ID)
}
