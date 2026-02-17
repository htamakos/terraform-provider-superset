// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
	"github.com/oapi-codegen/nullable"
)

var _ resource.Resource = &datasetColumnsResource{}
var _ resource.ResourceWithImportState = &datasetColumnsResource{}

func NewDatasetColumnsResource() resource.Resource {
	return &datasetColumnsResource{}
}

type datasetColumnsResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type datasetColumnsResourceModel struct {
	datasetColumnsBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *datasetColumnsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_columns"
}

func (r *datasetColumnsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset Dataset Columns",

		Attributes: map[string]schema.Attribute{
			"dataset_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The database ID of the datasetColumns.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The dataset name of the datasetColumns.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"columns": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "The columns of the dataset.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The column ID.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseNonNullStateForUnknown(),
							},
						},
						"advanced_data_type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The advanced data type of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"column_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the column.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The description of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"expression": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The expression of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"certified_by": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The user who certified the column.",
						},
						"certification_details": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The details of the column certification.",
						},
						"filterable": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the column is filterable.",
						},
						"groupby": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the column is groupable.",
						},
						"is_active": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the column is active.",
						},
						"is_dttm": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether the column is a datetime column.",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The data type of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"verbose_name": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The verbose name of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true, Update: true, Delete: true,
			}),
		},
	}
}

func (r *datasetColumnsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *datasetColumnsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data datasetColumnsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	_dataset, err := r.client.FindDataset(ctx, data.DatasetName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find dataset with name '%s': %s", data.DatasetName.ValueString(), err))
		return
	}
	dataset, err := r.client.GetDataset(ctx, _dataset.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get dataset with ID %d: %s", dataset.Id, err))
		return
	}

	columns := data.resovleColumns(dataset.Columns)

	putData := client.DatasetRestApiPut{}

	var datasetColumns []client.DatasetColumnsPut
	for _, column := range columns {
		datasetColumn := client.DatasetColumnsPut{
			Id:         int(column.Id.ValueInt64()),
			ColumnName: column.ColumnName.ValueString(),
			Filterable: column.Filterable.ValueBool(),
			Groupby:    column.Groupby.ValueBool(),
			IsActive:   nullable.NewNullableWithValue(column.IsActive.ValueBool()),
			IsDttm:     nullable.NewNullableWithValue(column.IsDttm.ValueBool()),
			Type:       nullable.NewNullableWithValue(column.Type.ValueString()),
		}
		if !column.AdvancedDataType.IsNull() && column.AdvancedDataType.ValueString() != "" {
			datasetColumn.AdvancedDataType = nullable.NewNullableWithValue(column.AdvancedDataType.ValueString())
		}
		if !column.Description.IsNull() && column.Description.ValueString() != "" {
			datasetColumn.Description = nullable.NewNullableWithValue(column.Description.ValueString())
		}
		if !column.Expression.IsNull() && column.Expression.ValueString() != "" {
			datasetColumn.Expression = nullable.NewNullableWithValue(column.Expression.ValueString())
		}
		if !column.VerboseName.IsNull() && column.VerboseName.ValueString() != "" {
			datasetColumn.VerboseName = nullable.NewNullableWithValue(column.VerboseName.ValueString())
		}
		if !column.CertifiedBy.IsNull() || column.CertifiedBy.ValueString() != "" {
			extra, err := column.toExtra()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert column certification info to extra field for column '%s': %s", column.ColumnName.ValueString(), err))
				return
			}
			datasetColumn.Extra = nullable.NewNullableWithValue(extra)
		}

		datasetColumns = append(datasetColumns, datasetColumn)
	}
	putData.Columns = datasetColumns

	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}
	if err := data.updateState(d); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state from dataset with ID %d: %s", dataset.Id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetColumnsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data datasetColumnsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	t, err := r.client.GetDataset(ctx, int(data.DatasetId.ValueInt64()))
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read datasetColumns with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	if err := data.updateState(t); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state from dataset with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetColumnsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state datasetColumnsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_dataset, err := r.client.FindDataset(ctx, plan.DatasetName.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find dataset with name '%s': %s", plan.DatasetName.ValueString(), err))
		return
	}

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()
	dataset, err := r.client.GetDataset(ctx, _dataset.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get dataset with ID %d: %s", dataset.Id, err))
		return
	}

	putData := client.DatasetRestApiPut{}

	resolvedColumns := plan.resovleColumns(dataset.Columns)
	var columns []client.DatasetColumnsPut
	stateColumnsMap := make(map[int64]datasetColumn)
	for _, column := range state.Columns {
		stateColumnsMap[column.Id.ValueInt64()] = column
	}

	for _, column := range resolvedColumns {
		_column := client.DatasetColumnsPut{
			Id:         int(column.Id.ValueInt64()),
			ColumnName: column.ColumnName.ValueString(),
			Filterable: column.Filterable.ValueBool(),
			Groupby:    column.Groupby.ValueBool(),
			IsActive:   nullable.NewNullableWithValue(column.IsActive.ValueBool()),
			IsDttm:     nullable.NewNullableWithValue(column.IsDttm.ValueBool()),
			Type:       nullable.NewNullableWithValue(stateColumnsMap[column.Id.ValueInt64()].Type.ValueString()),
		}
		if !column.AdvancedDataType.IsNull() {
			_column.AdvancedDataType = nullable.NewNullableWithValue(column.AdvancedDataType.ValueString())
		}
		if !column.Description.IsNull() {
			_column.Description = nullable.NewNullableWithValue(column.Description.ValueString())
		}
		if !column.Expression.IsNull() {
			_column.Expression = nullable.NewNullableWithValue(column.Expression.ValueString())
		}
		if !column.VerboseName.IsNull() {
			_column.VerboseName = nullable.NewNullableWithValue(column.VerboseName.ValueString())
		}
		if !column.CertifiedBy.IsNull() || column.CertifiedBy.ValueString() != "" {
			extra, err := column.toExtra()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert column certification info to extra field for column '%s': %s", column.ColumnName.ValueString(), err))
				return
			}
			_column.Extra = nullable.NewNullableWithValue(extra)
		}

		columns = append(columns, _column)
	}
	putData.Columns = columns

	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}

	if err := state.updateState(d); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update state from dataset with ID %d: %s", dataset.Id, err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *datasetColumnsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datasetColumnsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	// Delete is not supported for dataset columns, so we just update the dataset to remove the columns
	dataset, err := r.client.FindDataset(ctx, state.DatasetName.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find dataset with name '%s': %s", state.DatasetName.ValueString(), err))
		return
	}

	putData := client.DatasetRestApiPut{
		Columns: []client.DatasetColumnsPut{},
	}
	_, err = r.client.UpdateDataset(ctx, dataset.Id, putData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *datasetColumnsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	resp.State.SetAttribute(ctx, path.Root("dataset_id"), id)
}
