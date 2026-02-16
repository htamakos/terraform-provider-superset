// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type datasetMetricsBaseModel struct {
	DatasetId   types.Int64              `tfsdk:"dataset_id"`
	DatasetName types.String             `tfsdk:"dataset_name"`
	Metrics     map[string]datasetMetric `tfsdk:"metrics"`
}

var currencyAttrTypes = map[string]attr.Type{
	"symbol":          types.StringType,
	"symbol_position": types.StringType,
}

type datasetMetric struct {
	Id                   types.Int64  `tfsdk:"id"`
	Currency             types.Object `tfsdk:"currency"`
	D3format             types.String `tfsdk:"d3format"`
	Description          types.String `tfsdk:"description"`
	Expression           types.String `tfsdk:"expression"`
	CertifiedBy          types.String `tfsdk:"certified_by"`
	CertificationDetails types.String `tfsdk:"certification_details"`
	MetricName           types.String `tfsdk:"metric_name"`
	VerboseName          types.String `tfsdk:"verbose_name"`
	WarningText          types.String `tfsdk:"warning_text"`
}

type datasetMetricsExtra struct {
	Certification datasetMetricsExtraCertification `json:"certification"`
}

type datasetMetricsExtraCertification struct {
	CertifiedBy          string `json:"certified_by"`
	CertificationDetails string `json:"details"`
}

func (model *datasetMetric) toExtra() (string, error) {
	if model.CertifiedBy.IsNull() && model.CertificationDetails.IsNull() {
		return "", nil
	}

	extraData := datasetMetricsExtra{
		Certification: datasetMetricsExtraCertification{
			CertifiedBy:          model.CertifiedBy.ValueString(),
			CertificationDetails: model.CertificationDetails.ValueString(),
		},
	}

	extraBytes, err := json.Marshal(extraData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal extra data for metric '%s': %w", model.MetricName, err)
	}

	return string(extraBytes), nil
}

func (model *datasetMetric) toCurrency() (*client.DatasetMetricCurrencyPut, error) {
	if model.Currency.IsNull() {
		return nil, nil
	}

	attributes := model.Currency.Attributes()

	symbolAttr, ok := attributes["symbol"]
	if !ok {
		return nil, fmt.Errorf("currency object is missing 'symbol' attribute for metric '%s'", model.MetricName)
	}
	symbolPositionAttr, ok := attributes["symbol_position"]
	if !ok {
		return nil, fmt.Errorf("currency object is missing 'symbol_position' attribute for metric '%s'", model.MetricName)
	}

	symbolValue, ok := symbolAttr.(types.String)
	if !ok {

		return nil, fmt.Errorf("currency 'symbol' attribute is not a string for metric '%s'", model.MetricName)
	}
	symbolPositionValue, ok := symbolPositionAttr.(types.String)
	if !ok {
		return nil, fmt.Errorf("currency 'symbol_position' attribute is not a string for metric '%s'", model.MetricName)
	}

	return &client.DatasetMetricCurrencyPut{
		Symbol:         symbolValue.ValueString(),
		SymbolPosition: symbolPositionValue.ValueString(),
	}, nil
}

func (model *datasetMetric) parseCertification(d *client.DatasetRestApiGetSqlMetric) error {
	if !d.Extra.IsNull() && d.Extra.MustGet() != "" {
		var extraData datasetMetricsExtra

		if err := json.Unmarshal([]byte(d.Extra.MustGet()), &extraData); err != nil {
			return fmt.Errorf("failed to parse extra field for metric '%s': %w", model.MetricName, err)
		}

		if extraData.Certification.CertifiedBy != "" {
			model.CertifiedBy = types.StringValue(extraData.Certification.CertifiedBy)
		}
		if extraData.Certification.CertificationDetails != "" {
			model.CertificationDetails = types.StringValue(extraData.Certification.CertificationDetails)
		}
	}

	return nil
}

func (model *datasetMetricsBaseModel) updateState(d *client.DatasetRestApiGet) error {
	model.DatasetId = types.Int64Value(int64(d.Id))
	model.DatasetName = types.StringValue(d.TableName)

	metrics := make(map[string]datasetMetric)

	for _, metric := range d.Metrics {
		var m datasetMetric
		if err := m.updateState(&metric); err != nil {
			return err
		}
		metrics[metric.MetricName] = m
	}
	model.Metrics = metrics
	return nil
}

func (model *datasetMetric) updateState(d *client.DatasetRestApiGetSqlMetric) error {
	model.Id = types.Int64Value(int64(d.Id))
	if !d.D3format.IsNull() && d.D3format.MustGet() != "" {
		model.D3format = types.StringValue(d.D3format.MustGet())
	}
	if !d.Description.IsNull() && d.Description.MustGet() != "" {
		model.Description = types.StringValue(d.Description.MustGet())
	}
	model.Expression = types.StringValue(d.Expression)
	model.MetricName = types.StringValue(d.MetricName)
	if !d.VerboseName.IsNull() && d.VerboseName.MustGet() != "" {
		model.VerboseName = types.StringValue(d.VerboseName.MustGet())
	}
	if !d.WarningText.IsNull() && d.WarningText.MustGet() != "" {
		model.WarningText = types.StringValue(d.WarningText.MustGet())
	}
	if !d.Currency.IsNull() {
		attrValue, err := types.ObjectValue(currencyAttrTypes, map[string]attr.Value{
			"symbol":          types.StringValue(d.Currency.MustGet().Symbol),
			"symbol_position": types.StringValue(d.Currency.MustGet().SymbolPosition),
		})
		if err != nil {
			return fmt.Errorf("failed to create currency object for metric '%s'", d.MetricName)
		}

		model.Currency = attrValue
	} else {
		model.Currency = types.ObjectNull(currencyAttrTypes)
	}

	if err := model.parseCertification(d); err != nil {
		return err
	}

	return nil
}
