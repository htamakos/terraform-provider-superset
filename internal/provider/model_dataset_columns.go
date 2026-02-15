// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

type datasetColumnsBaseModel struct {
	DatasetId   types.Int64     `tfsdk:"dataset_id"`
	DatasetName types.String    `tfsdk:"dataset_name"`
	Columns     []datasetColumn `tfsdk:"columns"`
}

type datasetColumn struct {
	Id               types.Int64  `tfsdk:"id"`
	AdvancedDataType types.String `tfsdk:"advanced_data_type"`
	ColumnName       types.String `tfsdk:"column_name"`
	Description      types.String `tfsdk:"description"`
	Expression       types.String `tfsdk:"expression"`
	Extra            types.String `tfsdk:"extra"`
	Filterable       types.Bool   `tfsdk:"filterable"`
	Groupby          types.Bool   `tfsdk:"groupby"`
	IsActive         types.Bool   `tfsdk:"is_active"`
	IsDttm           types.Bool   `tfsdk:"is_dttm"`
	Type             types.String `tfsdk:"type"`
	VerboseName      types.String `tfsdk:"verbose_name"`
}

func (model *datasetColumnsBaseModel) updateState(d *client.DatasetRestApiGet) {
	model.DatasetId = types.Int64Value(int64(d.Id))
	model.DatasetName = types.StringValue(d.TableName)
	columns := make([]datasetColumn, len(d.Columns))
	for i, column := range d.Columns {
		var c datasetColumn
		c.updateState(&column)
		columns[i] = c
	}
	model.Columns = columns
}

func (model *datasetColumn) updateState(d *client.DatasetRestApiGetTableColumn) {
	model.Id = types.Int64Value(int64(d.Id))
	if d.AdvancedDataType.IsNull() || d.AdvancedDataType.MustGet() == "" {
		model.AdvancedDataType = types.StringNull()
	} else {
		model.AdvancedDataType = types.StringValue(d.AdvancedDataType.MustGet())
	}
	model.ColumnName = types.StringValue(d.ColumnName)
	if d.Description.IsNull() || d.Description.MustGet() == "" {
		model.Description = types.StringNull()
	} else {
		model.Description = types.StringValue(d.Description.MustGet())
	}
	if d.Expression.IsNull() || d.Expression.MustGet() == "" {
		model.Expression = types.StringNull()
	} else {
		model.Expression = types.StringValue(d.Expression.MustGet())
	}
	if d.Extra.IsNull() || d.Extra.MustGet() == "" {
		model.Extra = types.StringNull()
	} else {
		model.Extra = types.StringValue(d.Extra.MustGet())
	}
	if !d.Filterable.IsNull() {
		model.Filterable = types.BoolValue(d.Filterable.MustGet())
	} else {
		model.Filterable = types.BoolNull()
	}
	if !d.Groupby.IsNull() {
		model.Groupby = types.BoolValue(d.Groupby.MustGet())
	} else {
		model.Groupby = types.BoolNull()
	}
	if !d.IsActive.IsNull() {
		model.IsActive = types.BoolValue(d.IsActive.MustGet())
	} else {
		model.IsActive = types.BoolNull()
	}
	if !d.IsDttm.IsNull() {
		model.IsDttm = types.BoolValue(d.IsDttm.MustGet())
	} else {
		model.IsDttm = types.BoolNull()
	}
	if d.Type.IsNull() || d.Type.MustGet() == "" {
		model.Type = types.StringNull()
	} else {
		model.Type = types.StringValue(d.Type.MustGet())
	}
	if d.VerboseName.IsNull() || d.VerboseName.MustGet() == "" {
		model.VerboseName = types.StringNull()
	} else {
		model.VerboseName = types.StringValue(d.VerboseName.MustGet())
	}
}

func (model *datasetColumnsBaseModel) resovleColumns(columns []client.DatasetRestApiGetTableColumn) []datasetColumn {
	var resolvedColumns []datasetColumn

	mapColumnNameToColumn := make(map[string]client.DatasetRestApiGetTableColumn)
	for _, c := range columns {
		mapColumnNameToColumn[c.ColumnName] = c
	}

	for _, column := range model.Columns {
		columnName := column.ColumnName.ValueString()
		if c, ok := mapColumnNameToColumn[columnName]; ok {
			column.Id = types.Int64Value(int64(c.Id))
			if !c.Type.IsNull() && c.Type.MustGet() != "" {
				column.Type = types.StringValue(c.Type.MustGet())
			}
			resolvedColumns = append(resolvedColumns, column)
		} else {
			resolvedColumns = append(resolvedColumns, column)
		}
	}

	return resolvedColumns
}
