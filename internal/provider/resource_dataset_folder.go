// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

var _ resource.Resource = &datasetFolderResource{}
var _ resource.ResourceWithImportState = &datasetFolderResource{}

func NewDatasetFolderResource() resource.Resource {
	return &datasetFolderResource{}
}

type datasetFolderResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type datasetFolderResourceModel struct {
	datasetFolderBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *datasetFolderResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_folder"
}

func (r *datasetFolderResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset Dataset folder",

		Attributes: map[string]schema.Attribute{
			"dataset_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The database ID of the datasetfolder.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The dataset name of the datasetfolder.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"folders": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "The folder of the dataset.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The folder Name.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"type": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The folder type.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("folder"),
							},
						},
						"description": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The description of the column.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"uuid": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The UUID of the folder.",
						},
						"children": schema.ListNestedAttribute{
							Required:            true,
							MarkdownDescription: "The children of the folder.",
							Validators: []validator.List{
								listvalidator.SizeAtLeast(1),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The child Name.",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The child type.",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
										Validators: []validator.String{
											stringvalidator.OneOf("column", "metric"),
										},
									},
									"description": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "The description of the child.",
										PlanModifiers: []planmodifier.String{
											stringplanmodifier.UseStateForUnknown(),
										},
									},
									"uuid": schema.StringAttribute{
										Computed:            true,
										MarkdownDescription: "The UUID of the folder.",
									},
								},
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

func (r *datasetFolderResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *datasetFolderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data datasetFolderResourceModel

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

	data.resolveColumns(dataset.Columns)
	folders, err := data.toFolders()
	if err != nil {
		resp.Diagnostics.AddError("Folder Conversion Error", fmt.Sprintf("Unable to convert folders for dataset with ID %d: %s", dataset.Id, err))
		return
	}

	putData := client.DatasetRestApiPut{
		Folders: folders,
	}

	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}
	if err := data.updateState(d); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state for datasetfolder with ID %d: %s", dataset.Id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetFolderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data datasetFolderResourceModel

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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read datasetfolder with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	if err := data.updateState(t); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state for datasetfolder with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetFolderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state datasetFolderResourceModel

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

	plan.resolveColumns(dataset.Columns)
	folders, err := plan.toFolders()
	if err != nil {
		resp.Diagnostics.AddError("Folder Conversion Error", fmt.Sprintf("Unable to convert folders for dataset with ID %d: %s", dataset.Id, err))
		return
	}

	putData := client.DatasetRestApiPut{
		Folders: folders,
	}

	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}

	if err := state.updateState(d); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state for datasetfolder with ID %d: %s", dataset.Id, err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *datasetFolderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datasetFolderResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	// Delete is not supported for dataset folder, so we just update the dataset to remove the folder
	dataset, err := r.client.FindDataset(ctx, state.DatasetName.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find dataset with name '%s': %s", state.DatasetName.ValueString(), err))
		return
	}

	putData := client.DatasetRestApiPut{
		Folders: []client.Folder{},
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

func (r *datasetFolderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
