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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
    "github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

var _ resource.Resource = &RolePermissionsResource{}
var _ resource.ResourceWithImportState = &RolePermissionsResource{}

func NewRolePermissionsResource() resource.Resource {
	return &RolePermissionsResource{}
}

type RolePermissionsResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type rolePermissionsResourceModel struct {
	rolePermissionBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *RolePermissionsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_permissions"
}

func (r *RolePermissionsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset role with permissions",

		Attributes: map[string]schema.Attribute{
            "role_id": schema.Int64Attribute{
                Computed:            true,
                MarkdownDescription: "The ID of the role.",
                PlanModifiers: []planmodifier.Int64{
                    int64planmodifier.UseStateForUnknown(),
                },
            },
			"role_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
            "permissions": schema.ListNestedAttribute{
                Required:            true,
                Validators: []validator.List{
                  listvalidator.SizeAtLeast(1),
                },
                NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "permission_name": schema.StringAttribute{
                            Required:            true,
                            MarkdownDescription: "The name of the permission.",
                        },
                        "view_menu_name": schema.StringAttribute{
                            Required:            true,
                            MarkdownDescription: "The name of the view menu.",
                        },
                    },
                },
                MarkdownDescription: "The list of permissions assigned to the role.",
            },
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, Update: true, Delete: true,
			}),
		},
	}
}

func (r *RolePermissionsResource) ValidateConfig(
    ctx context.Context,
    req resource.ValidateConfigRequest,
    resp *resource.ValidateConfigResponse,
) {
    var data rolePermissionsResourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    if data.Permissions.IsNull() || data.Permissions.IsUnknown() {
        return
    }

    elems := data.Permissions.Elements()
    seen := make(map[string]int, len(elems))

    for i, v := range elems {
        obj, ok := v.(types.Object)
        if !ok {
            resp.Diagnostics.AddAttributeError(
                path.Root("permissions").AtListIndex(i),
                "Invalid permission element",
                "Each permissions element must be an object.",
            )
            continue
        }

        pn := obj.Attributes()["permission_name"].(types.String)
        vm := obj.Attributes()["view_menu_name"].(types.String)

        if pn.IsNull() || pn.IsUnknown() || vm.IsNull() || vm.IsUnknown() {
            resp.Diagnostics.AddAttributeError(
                path.Root("permissions").AtListIndex(i),
                "Permission is not fully specified",
                "permission_name and view_menu_name must be set.",
            )
            continue
        }

        key := pn.ValueString() + "_" + vm.ValueString()
        if firstIdx, exists := seen[key]; exists {
            resp.Diagnostics.AddAttributeError(
                path.Root("permissions").AtListIndex(i),
                "Duplicate permission",
                fmt.Sprintf("Duplicate permission %q. It was already specified at permissions[%d].", key, firstIdx),
            )
            continue
        }
        seen[key] = i
    }
}

func (r *RolePermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RolePermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data rolePermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    role, err := r.client.FindRole(ctx, data.RoleName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find role with name %s: %s", data.RoleName.ValueString(), err))
        return
    }
    permissionIds := make([]int, 0, len(data.Permissions.Elements()))
    sourcePermissions, err := r.client.ListPermissions(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list permissions: %s", err))
        return
    }

    permissions, notFoundPermissions := data.resolvePermissions(sourcePermissions)
    if len(notFoundPermissions) > 0 {
        resp.Diagnostics.AddError("Invalid Permissions", fmt.Sprintf("The following permissions were not found: %v", notFoundPermissions))
        return
    }
    for _, permission := range permissions {
        permissionIds = append(permissionIds, permission.Id)
    }

	err = r.client.AssignPermissionsToRole(ctx, role.Id, permissionIds)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err))
		return
	}

	data.updateState(int64(role.Id), role.Name, permissions)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RolePermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data rolePermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    role, err := r.client.FindRole(ctx, data.RoleName.ValueString())
    if client.IsNotFound(err) {
        resp.State.RemoveResource(ctx)
        return
    } else if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find role with name %s: %s", data.RoleName.ValueString(), err))
        return
    }

    permissions, err := r.client.ListRolePermissions(ctx, role.Id)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list permissions for role ID %d: %s", role.Id, err))
        return
    }

	data.updateState(int64(role.Id), role.Name, permissions)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RolePermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state rolePermissionsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    role, err := r.client.FindRole(ctx, plan.RoleName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find role with name %s: %s", plan.RoleName.ValueString(), err))
        return
    }
    permissionIds := make([]int, 0, len(plan.Permissions.Elements()))
    sourcePermissions, err := r.client.ListPermissions(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list permissions: %s", err))
        return
    }
    permissions, notFoundPermissions := plan.resolvePermissions(sourcePermissions)
    if len(notFoundPermissions) > 0 {
        resp.Diagnostics.AddError("Invalid Permissions", fmt.Sprintf("The following permissions were not found: %v", notFoundPermissions))
        return
    }
    for _, permission := range permissions {
        permissionIds = append(permissionIds, permission.Id)
    }

	err = r.client.AssignPermissionsToRole(ctx, role.Id, permissionIds)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role, got error: %s", err))
		return
	}

	state.updateState(int64(role.Id), role.Name, permissions)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RolePermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rolePermissionsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

    role, err := r.client.FindRole(ctx, state.RoleName.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find role with name %s: %s", state.RoleName.ValueString(), err))
        return
    }

    err = r.client.AssignPermissionsToRole(ctx, role.Id, []int{})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role with ID %d: %s", role.Id, err))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RolePermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Starting ImportState method", map[string]interface{}{
		"import_id": req.ID,
	})

	resp.State.SetAttribute(ctx, path.Root("role_name"), req.ID)
}
