// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type datasetBaseModel struct {
	Id                    types.Int64  `tfsdk:"id"`
	DatabaseId            types.Int64  `tfsdk:"database_id"`
	DatabaseName          types.String `tfsdk:"database_name"`
	BootstrapDatabaseId   types.Int64  `tfsdk:"bootstrap_database_id"`
	BootstrapDatabaseName types.String `tfsdk:"bootstrap_database_name"`
	Catalog               types.String `tfsdk:"catalog"`
	Schema                types.String `tfsdk:"schema"`
	TableName             types.String `tfsdk:"table_name"`
	Sql                   types.String `tfsdk:"sql"`
	Description           types.String `tfsdk:"description"`
	CacheTimeout          types.Int64  `tfsdk:"cache_timeout"`
	IsManagedExternally   types.Bool   `tfsdk:"is_managed_externally"`
	FilterSelectEnabled   types.Bool   `tfsdk:"filter_select_enabled"`
	FetchValuesPredicate  types.String `tfsdk:"fetch_values_predicate"`
	AlwaysFilterMainDttm  types.Bool   `tfsdk:"always_filter_main_dttm"`
	NormalizeColumns      types.Bool   `tfsdk:"normalize_columns"`
	OwnerIds              types.Set    `tfsdk:"owner_ids"`
	CertifiedBy           types.String `tfsdk:"certified_by"`
	CertificationDetails  types.String `tfsdk:"certification_details"`
}

type datasetExtra struct {
	Certification datasetExtraCertification `json:"certification"`
}

type datasetExtraCertification struct {
	CertifiedBy          string `json:"certified_by"`
	CertificationDetails string `json:"details"`
}

func (model *datasetBaseModel) toExtra() (string, error) {
	if model.CertifiedBy.IsNull() && model.CertificationDetails.IsNull() {
		return "", nil
	}

	extraData := datasetExtra{
		Certification: datasetExtraCertification{
			CertifiedBy:          model.CertifiedBy.ValueString(),
			CertificationDetails: model.CertificationDetails.ValueString(),
		},
	}

	extraBytes, err := json.Marshal(extraData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal extra data for column '%s': %w", model.TableName, err)
	}

	return string(extraBytes), nil
}

func (model *datasetBaseModel) parseCertification(d *client.DatasetRestApiGet) error {
	if !d.Extra.IsNull() && d.Extra.MustGet() != "" {
		var extraData datasetColumnExtra

		if err := json.Unmarshal([]byte(d.Extra.MustGet()), &extraData); err != nil {
			return fmt.Errorf("failed to parse extra field for column '%s': %w", model.TableName, err)
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

func (model *datasetBaseModel) updateState(d *client.DatasetRestApiGet) error {
	model.Id = types.Int64Value(int64(d.Id))
	model.DatabaseId = types.Int64Value(int64(d.Database.Id))
	model.DatabaseName = types.StringValue(d.Database.DatabaseName)

	model.TableName = types.StringValue(d.TableName)
	if d.Sql.IsNull() || d.Sql.MustGet() == "" {
		model.Sql = types.StringNull()
	} else {
		model.Sql = types.StringValue(d.Sql.MustGet())
	}
	if d.Description.IsNull() || d.Description.MustGet() == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(d.Description.MustGet())
	}
	if d.CacheTimeout.IsNull() {
		model.CacheTimeout = types.Int64Null()
	} else {
		model.CacheTimeout = types.Int64Value(int64(d.CacheTimeout.MustGet()))
	}
	if !d.FilterSelectEnabled.IsNull() {
		model.FilterSelectEnabled = types.BoolValue(d.FilterSelectEnabled.MustGet())
	} else {
		model.FilterSelectEnabled = types.BoolNull()
	}

	if !d.FetchValuesPredicate.IsNull() && d.FetchValuesPredicate.MustGet() != "" {
		model.FetchValuesPredicate = types.StringValue(d.FetchValuesPredicate.MustGet())
	}

	if !d.AlwaysFilterMainDttm.IsNull() {
		model.AlwaysFilterMainDttm = types.BoolValue(d.AlwaysFilterMainDttm.MustGet())
	} else {
		model.AlwaysFilterMainDttm = types.BoolNull()
	}
	if !d.NormalizeColumns.IsNull() {
		model.NormalizeColumns = types.BoolValue(d.NormalizeColumns.MustGet())
	} else {
		model.NormalizeColumns = types.BoolNull()
	}

	model.IsManagedExternally = types.BoolValue(d.IsManagedExternally)

	if err := model.parseCertification(d); err != nil {
		return err
	}

	return nil
}
