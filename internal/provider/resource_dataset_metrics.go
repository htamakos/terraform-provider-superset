// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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
	"github.com/oapi-codegen/nullable"
)

var _ resource.Resource = &datasetMetricsResource{}
var _ resource.ResourceWithImportState = &datasetMetricsResource{}

func NewDatasetMetricsResource() resource.Resource {
	return &datasetMetricsResource{}
}

type datasetMetricsResource struct {
	client   *client.ClientWrapper
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type datasetMetricsResourceModel struct {
	datasetMetricsBaseModel
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *datasetMetricsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_metrics"
}

func (r *datasetMetricsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage a superset Dataset metrics",

		Attributes: map[string]schema.Attribute{
			"dataset_id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "The database ID of the datasetmetrics.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"dataset_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The dataset name of the datasetmetrics.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metrics": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "The metrics of the dataset.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "The metric ID.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseNonNullStateForUnknown(),
							},
						},
						"currency": schema.SingleNestedAttribute{
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"symbol": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.OneOf("GBP", "USD", "JPY", "INR", "CNY", "MXN"),
									},
								},
								"symbol_position": schema.StringAttribute{
									Required: true,
									Validators: []validator.String{
										stringvalidator.OneOf("prefix", "suffix"),
									},
								},
							},
						},
						"d3format": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The D3 format of the metric.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"expression": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The expression of the metric.",
						},
						"description": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The description of the metric.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"certified_by": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The name of the person or organization that certified the metric.",
						},
						"certification_details": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The details of the metric certification.",
						},
						"metric_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the metric.",
						},
						"verbose_name": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The verbose name of the metric.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"warning_text": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The warning text of the metric.",
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

func (r *datasetMetricsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *datasetMetricsResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var data datasetMetricsResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for key, m := range data.Metrics {
		if m.MetricName.IsUnknown() || m.MetricName.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("metrics").AtMapKey(key).AtName("metric_name"),
				"metric_name is required",
				"The metric_name attribute is required for each metric and cannot be unknown or null.",
			)
			continue
		}

		v := m.MetricName.ValueString()
		if v != key {
			resp.Diagnostics.AddAttributeError(
				path.Root("metrics").AtMapKey(key).AtName("metric_name"),
				"metrics key must match metric_name",
				fmt.Sprintf("The key '%s' in the metrics map must match the metric_name '%s'.", key, v),
			)
		}
	}
}

func (r *datasetMetricsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data datasetMetricsResourceModel

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

	putData := client.DatasetRestApiPut{}
	var datasetMetrics []client.DatasetMetricsPut

	for _, metric := range data.Metrics {
		datasetMetric := client.DatasetMetricsPut{
			Id:         int(metric.Id.ValueInt64()),
			MetricName: metric.MetricName.ValueString(),
			Expression: metric.Expression.ValueString(),
		}
		if !metric.Description.IsNull() && metric.Description.ValueString() != "" {
			datasetMetric.Description = nullable.NewNullableWithValue(metric.Description.ValueString())
		}
		if !metric.VerboseName.IsNull() && metric.VerboseName.ValueString() != "" {
			datasetMetric.VerboseName = nullable.NewNullableWithValue(metric.VerboseName.ValueString())
		}
		if !metric.D3format.IsNull() && metric.D3format.ValueString() != "" {
			datasetMetric.D3format = nullable.NewNullableWithValue(metric.D3format.ValueString())
		}
		if !metric.WarningText.IsNull() && metric.WarningText.ValueString() != "" {
			datasetMetric.WarningText = nullable.NewNullableWithValue(metric.WarningText.ValueString())
		}

		if !metric.Currency.IsNull() {
			currency, err := metric.toCurrency()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert currency for metric '%s': %s", metric.MetricName.ValueString(), err))
				return
			}
			datasetMetric.Currency = nullable.NewNullableWithValue(*currency)
		}
		if !metric.CertifiedBy.IsNull() || !metric.CertificationDetails.IsNull() {
			extra, err := metric.toExtra()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert extra data for metric '%s': %s", metric.MetricName.ValueString(), err))
				return
			}
			datasetMetric.Extra = nullable.NewNullableWithValue(extra)
		}

		datasetMetrics = append(datasetMetrics, datasetMetric)
	}
	putData.Metrics = datasetMetrics

	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}
	if err := data.updateState(d); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state from API response for dataset with ID %d: %s", dataset.Id, err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetMetricsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data datasetMetricsResourceModel

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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read datasetmetrics with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	if err := data.updateState(t); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state from API response for dataset with ID %d: %s", data.DatasetId.ValueInt64(), err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *datasetMetricsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state datasetMetricsResourceModel

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

	var datasetMetrics []client.DatasetMetricsPut

	for _, metric := range plan.Metrics {
		datasetMetric := client.DatasetMetricsPut{
			Id:         int(metric.Id.ValueInt64()),
			MetricName: metric.MetricName.ValueString(),
			Expression: metric.Expression.ValueString(),
		}
		if !metric.Description.IsNull() && metric.Description.ValueString() != "" {
			datasetMetric.Description = nullable.NewNullableWithValue(metric.Description.ValueString())
		}
		if !metric.VerboseName.IsNull() && metric.VerboseName.ValueString() != "" {
			datasetMetric.VerboseName = nullable.NewNullableWithValue(metric.VerboseName.ValueString())
		}
		if !metric.D3format.IsNull() && metric.D3format.ValueString() != "" {
			datasetMetric.D3format = nullable.NewNullableWithValue(metric.D3format.ValueString())
		}
		if !metric.WarningText.IsNull() && metric.WarningText.ValueString() != "" {
			datasetMetric.WarningText = nullable.NewNullableWithValue(metric.WarningText.ValueString())
		}
		if !metric.Currency.IsNull() {
			currency, err := metric.toCurrency()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert currency for metric '%s': %s", metric.MetricName.ValueString(), err))
				return
			}
			datasetMetric.Currency = nullable.NewNullableWithValue(*currency)
		}
		if !metric.CertifiedBy.IsNull() || !metric.CertificationDetails.IsNull() {
			extra, err := metric.toExtra()
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert extra data for metric '%s': %s", metric.MetricName.ValueString(), err))
				return
			}
			datasetMetric.Extra = nullable.NewNullableWithValue(extra)
		}

		datasetMetrics = append(datasetMetrics, datasetMetric)
	}
	putData.Metrics = datasetMetrics
	d, err := r.client.UpdateDataset(ctx, dataset.Id, putData)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update dataset with ID %d: %s", dataset.Id, err))
		return
	}

	if err := state.updateState(d); err != nil {
		resp.Diagnostics.AddError("State Update Error", fmt.Sprintf("Unable to update state from API response for dataset with ID %d: %s", dataset.Id, err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *datasetMetricsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state datasetMetricsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	ctx, cancel := SetupTimeoutCreate(ctx, r.Timeouts, Timeout5min)
	defer cancel()

	// Delete is not supported for dataset metrics, so we just update the dataset to remove the metrics
	dataset, err := r.client.FindDataset(ctx, state.DatasetName.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	} else if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to find dataset with name '%s': %s", state.DatasetName.ValueString(), err))
		return
	}

	putData := client.DatasetRestApiPut{
		Metrics: []client.DatasetMetricsPut{},
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

func (r *datasetMetricsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
