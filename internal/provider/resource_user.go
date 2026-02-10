// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type userResourceModel struct {
	userBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset user",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the user.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The email of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The first name of the user.",
			},
			"last_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The last name of the user.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The password of the user.",
			},
			"role_names": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Role names to assign to the user.",
			},
			"group_names": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: setdefault.StaticValue(
					types.SetValueMust(types.StringType, []attr.Value{}),
				),
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Group names to assign to the user.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the user is active.",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, Update: true, Delete: true,
			}),
		},
	}
}

func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data userResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()
	postData := client.SupersetUserApiPost{
		Username:  data.Username.ValueString(),
		Email:     data.Email.ValueString(),
		FirstName: data.FirstName.ValueString(),
		LastName:  data.LastName.ValueString(),
		Password:  data.Password.ValueString(),
		Active:    data.Active.ValueBool(),
	}

	roles, err := r.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}
	roleIds, notFoundRoles := data.resolveRoleIDsFromNames(roles)
	if len(notFoundRoles) > 0 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find roles: %v", notFoundRoles))
		return
	}

	postData.Roles = roleIds

	if len(data.GroupNames.Elements()) > 0 {
		groups, err := r.client.ListGroups(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list groups: %s", err))
			return
		}
		groupIds, notFoundGroups := data.resolveGroupIDsFromNames(groups)

		if len(notFoundGroups) > 0 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find groups: %v", notFoundGroups))
			return
		}

		postData.Groups = groupIds
	}

	existingUser, err := r.client.FindUser(ctx, postData.Username)
	if !client.IsNotFound(err) && err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to validate user name uniqueness: %s", err))
		return
	}
	if existingUser != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("A user with username '%s' already exists with ID %d", postData.Username, existingUser.Id))
		return
	}

	u, err := r.client.CreateUser(ctx, postData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create user, got error: %s", err))
		return
	}
	var password *string
	if !data.Password.IsNull() {
		passwordValue := data.Password.ValueString()
		password = &passwordValue
	}

	data.updateState(u, password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	u, err := r.client.GetUser(ctx, int(state.Id.ValueInt64()))
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read user with ID %d: %s", state.Id.ValueInt64(), err))
		return
	}

	var password *string = nil
	if !state.Password.IsNull() {
		passwordValue := state.Password.ValueString()
		password = &passwordValue
	}

	state.updateState(u, password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state userResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	putData := client.SupersetUserApiPut{
		Email:     plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Password:  plan.Password.ValueString(),
		Active:    plan.Active.ValueBool(),
	}

	if len(plan.GroupNames.Elements()) > 0 {
		sourceGroups, err := r.client.ListGroups(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list groups: %s", err))
			return
		}
		groupIds, notFoundGroups := plan.resolveGroupIDsFromNames(sourceGroups)
		if len(notFoundGroups) > 0 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find groups: %v", notFoundGroups))
			return
		}
		putData.Groups = groupIds
	} else {
		putData.Groups = []int{}
	}

	if len(plan.RoleNames.Elements()) > 0 {
		roles, err := r.client.ListRoles(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
			return
		}
		roleIds, notFoundRoles := plan.resolveRoleIDsFromNames(roles)
		if len(notFoundRoles) > 0 {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find roles: %v", notFoundRoles))
			return
		}
		putData.Roles = roleIds
	} else {
		putData.Roles = []int{}
	}

	u, err := r.client.UpdateUser(ctx, int(plan.Id.ValueInt64()), putData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update user with ID %d: %s", state.Id.ValueInt64(), err))
		return
	}

	var password *string
	if !plan.Password.IsNull() {
		passwordValue := plan.Password.ValueString()
		password = &passwordValue
	}

	state.updateState(u, password)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	err := r.client.DeleteUser(ctx, int(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddWarning("Deletion Error", fmt.Sprintf("Unable to delete user with ID %d: %s", state.Id.ValueInt64(), err))

		_, err = r.client.UpdateUser(ctx, int(state.Id.ValueInt64()), client.SupersetUserApiPut{
			Active: false,
			Groups: []int{},
		})

		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to deactivate user with ID %d:, so deactivate user. error: %s", state.Id.ValueInt64(), err))
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Starting ImportState method", map[string]interface{}{
		"import_id": req.ID,
	})

	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected numeric ID, got %q: %s", req.ID, err),
		)
		return
	}

	resp.State.SetAttribute(ctx, path.Root("id"), id)
}
