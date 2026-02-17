// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
	"github.com/oapi-codegen/nullable"
)

var _ resource.Resource = &DatasetResource{}
var _ resource.ResourceWithImportState = &DatasetResource{}

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

type DatasetResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type DatasetResourceModel struct {
	datasetBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *DatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset Dataset",

		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The ID of the Dataset.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"database_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The database ID of the Dataset.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"bootstrap_database_name": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: `The database name of the Dataset used for bootstrapping.
Some Superset databases configured with OAuth authentication cannot be directly referenced during dataset creation via the Terraform provider, resulting in creation failures.

To mitigate this limitation, a temporary non-OAuth database is specified at creation time. Once the dataset resource is successfully created, it is immediately updated to reference the intended OAuth-authenticated database.

This database is not intended for operational use and exists solely to satisfy creation-time constraints.`,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bootstrap_database_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The database ID of the Dataset used for bootstrapping.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The database name of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"table_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The catalog of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schema": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The schema of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sql": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The SQL of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The description of the Dataset.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cache_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "The cache timeout of the Dataset.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"filter_select_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The filter select enabled of the Dataset.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Default: booldefault.StaticBool(false),
			},
			"fetch_values_predicate": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The fetch values predicate of the Dataset.",
			},
			"always_filter_main_dttm": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The always filter main dttm of the Dataset.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Default: booldefault.StaticBool(false),
			},
			"normalize_columns": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The normalize columns of the Dataset.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Default: booldefault.StaticBool(false),
			},
			"is_managed_externally": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the Dataset is managed externally.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				Default: booldefault.StaticBool(false),
			},
			"owner_ids": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "The owner IDs of the Dataset.",
				ElementType:         types.Int64Type,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"certified_by": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The user who certified the Dataset.",
			},
			"certification_details": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The details of the Dataset certification.",
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, Update: true, Delete: true,
			}),
		},
	}
}

func (r *DatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DatasetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	var bootstrapDatabaseName string
	if !data.BootstrapDatabaseName.IsNull() && data.BootstrapDatabaseName.ValueString() != "" {
		bootstrapDatabaseName = data.BootstrapDatabaseName.ValueString()
	} else {
		bootstrapDatabaseName = data.DatabaseName.ValueString()
	}

	database, err := r.client.FindDatabase(ctx, bootstrapDatabaseName)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find Database with name '%s': %s", data.DatabaseName.ValueString(), err))
		return
	}
	bootstrapDatabaseId := database.Id

	postData := client.DatasetRestApiPost{
		TableName:           data.TableName.ValueString(),
		Database:            database.Id,
		IsManagedExternally: nullable.NewNullableWithValue(data.IsManagedExternally.ValueBool()),
	}

	if !data.Catalog.IsNull() && data.Catalog.ValueString() != "" {
		postData.Catalog = nullable.NewNullableWithValue(data.Catalog.ValueString())
	}
	if !data.Schema.IsNull() {
		postData.Schema = nullable.NewNullableWithValue(data.Schema.ValueString())
	}
	if !data.Sql.IsNull() {
		postData.Sql = nullable.NewNullableWithValue(data.Sql.ValueString())
	}
	if !data.NormalizeColumns.IsNull() {
		postData.NormalizeColumns = data.NormalizeColumns.ValueBool()
	}

	existingDataset, err := r.client.FindDataset(ctx, postData.TableName)
	if !client.IsNotFound(err) && err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to validate Dataset name uniqueness: %s", err))
		return
	}
	if existingDataset != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("A Dataset with name '%s' already exists with ID %d", postData.TableName, existingDataset.Id))
		return
	}

	d, err := r.client.CreateDataset(ctx, postData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Dataset, got error: %s", err))
		return
	}

	isChangedBootstrapDatabase := data.DatabaseName.ValueString() != bootstrapDatabaseName

	if !data.Description.IsNull() || !data.CacheTimeout.IsNull() || !data.FilterSelectEnabled.IsNull() || isChangedBootstrapDatabase || !data.CertifiedBy.IsNull() || !data.FetchValuesPredicate.IsNull() || !data.AlwaysFilterMainDttm.IsNull() {
		putData := client.DatasetRestApiPut{}
		if !data.Description.IsNull() {
			putData.Description = nullable.NewNullableWithValue(data.Description.ValueString())
		}
		if !data.CacheTimeout.IsNull() {
			putData.CacheTimeout = nullable.NewNullableWithValue(int(data.CacheTimeout.ValueInt64()))
		}
		if !data.FilterSelectEnabled.IsNull() {
			putData.FilterSelectEnabled = nullable.NewNullableWithValue(data.FilterSelectEnabled.ValueBool())
		}

		if !data.FetchValuesPredicate.IsNull() {
			putData.FetchValuesPredicate = nullable.NewNullableWithValue(data.FetchValuesPredicate.ValueString())
		}

		if !data.AlwaysFilterMainDttm.IsNull() {
			putData.AlwaysFilterMainDttm = data.AlwaysFilterMainDttm.ValueBool()
		}

		if isChangedBootstrapDatabase {
			database, err = r.client.FindDatabase(ctx, data.DatabaseName.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find Database with name '%s': %s", data.DatabaseName.ValueString(), err))
				return
			}

			putData.DatabaseId = database.Id
		}

		if !data.OwnerIds.IsNull() && len(data.OwnerIds.Elements()) > 0 {
			var ownerIds []int
			for _, v := range data.OwnerIds.Elements() {
				ownerIdValue, ok := v.(types.Int64)
				if !ok {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse owner ID: expected int64, got %T", v))
					return
				}

				ownerIds = append(ownerIds, int(ownerIdValue.ValueInt64()))
			}
			putData.Owners = ownerIds
		}

		if !data.CertifiedBy.IsNull() || data.CertifiedBy.ValueString() != "" {
			extra, err := data.toExtra()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal extra data for Dataset '%s': %s", data.TableName.ValueString(), err))
				return
			}
			putData.Extra = nullable.NewNullableWithValue(extra)
		}

		d, err = r.client.UpdateDataset(ctx, d.Id, putData)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Dataset with ID %d: %s", d.Id, err))
			return
		}
	}

	if err := data.updateState(d); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state for Dataset with ID %d: %s", d.Id, err))
		return
	}

	data.BootstrapDatabaseId = types.Int64Value(int64(bootstrapDatabaseId))
	if isChangedBootstrapDatabase {
		data.BootstrapDatabaseName = types.StringValue(bootstrapDatabaseName)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DatasetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()
	t, err := r.client.GetDataset(ctx, int(data.Id.ValueInt64()))
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Dataset with ID %d: %s", data.Id.ValueInt64(), err))
		return
	}

	if err := data.updateState(t); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state for Dataset with ID %d: %s", data.Id.ValueInt64(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DatasetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	putData := client.DatasetRestApiPut{}

	if !plan.Description.IsNull() {
		putData.Description = nullable.NewNullableWithValue(plan.Description.ValueString())
	}
	if !plan.CacheTimeout.IsNull() {
		putData.CacheTimeout = nullable.NewNullableWithValue(int(plan.CacheTimeout.ValueInt64()))
	}
	if !plan.FilterSelectEnabled.IsNull() {
		putData.FilterSelectEnabled = nullable.NewNullableWithValue(plan.FilterSelectEnabled.ValueBool())
	}
	if !plan.FetchValuesPredicate.IsNull() {
		putData.FetchValuesPredicate = nullable.NewNullableWithValue(plan.FetchValuesPredicate.ValueString())
	}

	if !plan.Sql.IsNull() {
		putData.Sql = nullable.NewNullableWithValue(plan.Sql.ValueString())
	}
	if !plan.NormalizeColumns.IsNull() {
		putData.NormalizeColumns = nullable.NewNullableWithValue(plan.NormalizeColumns.ValueBool())
	}
	if !plan.AlwaysFilterMainDttm.IsNull() {
		putData.AlwaysFilterMainDttm = plan.AlwaysFilterMainDttm.ValueBool()
	}
	if !plan.Catalog.IsNull() && plan.Catalog.ValueString() != "" {
		putData.Catalog = nullable.NewNullableWithValue(plan.Catalog.ValueString())
	}
	if !plan.Schema.IsNull() {
		putData.Schema = nullable.NewNullableWithValue(plan.Schema.ValueString())
	}
	if !plan.TableName.IsNull() {
		putData.TableName = nullable.NewNullableWithValue(plan.TableName.ValueString())
	}
	if !plan.OwnerIds.IsNull() && len(plan.OwnerIds.Elements()) > 0 {
		var ownerIds []int
		for _, v := range plan.OwnerIds.Elements() {
			ownerIdValue, ok := v.(types.Int64)
			if !ok {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse owner ID: expected int64, got %T", v))
				return
			}
			ownerIds = append(ownerIds, int(ownerIdValue.ValueInt64()))
		}
		putData.Owners = ownerIds
	}

	if !plan.CertifiedBy.IsNull() || plan.CertifiedBy.ValueString() != "" {
		extra, err := plan.toExtra()
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal extra data for Dataset '%s': %s", plan.TableName.ValueString(), err))
			return
		}
		putData.Extra = nullable.NewNullableWithValue(extra)
	}

	g, err := r.client.UpdateDataset(ctx, int(state.Id.ValueInt64()), putData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Dataset with ID %d: %s", state.Id.ValueInt64(), err))
		return
	}

	if err := state.updateState(g); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state for Dataset with ID %d: %s", state.Id.ValueInt64(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	err := r.client.DeleteDataset(ctx, int(state.Id.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Dataset with ID %d: %s", state.Id.ValueInt64(), err))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
