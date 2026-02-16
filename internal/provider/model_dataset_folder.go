// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/htamakos/terraform-provider-superset/internal/client"
	"github.com/oapi-codegen/nullable"
)

type datasetFolderBaseModel struct {
	DatasetId   types.Int64          `tfsdk:"dataset_id"`
	DatasetName types.String         `tfsdk:"dataset_name"`
	Folders     []datasetFolderModel `tfsdk:"folders"`
}

type datasetFolderModel struct {
	Name        types.String              `tfsdk:"name"`
	Description types.String              `tfsdk:"description"`
	Type        types.String              `tfsdk:"type"`
	Children    []datasetFolderChildModel `tfsdk:"children"`
	Uuid        types.String              `tfsdk:"uuid"`
}

type datasetFolderChildModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Uuid        types.String `tfsdk:"uuid"`
}

func (model *datasetFolderBaseModel) updateState(d *client.DatasetRestApiGet) error {
	model.DatasetId = types.Int64Value(int64(d.Id))
	model.DatasetName = types.StringValue(d.TableName)

	if d.Folders.IsNull() {
		return nil
	}

	_folders := d.Folders.MustGet()
	typedFolders, err := mapToFolders(_folders)
	if err != nil {
		return fmt.Errorf("failed to map folders from API response: %w", err)
	}

	var folders []datasetFolderModel
	for _, folder := range typedFolders {
		var folderModel datasetFolderModel
		folderModel.Name = types.StringValue(folder.Name)

		if folder.Description.IsSpecified() && folder.Description.MustGet() != "" {
			folderModel.Description = types.StringValue(folder.Description.MustGet())
		}
		if folder.Type != client.FolderTypeFolder {
			return errors.New("unexpected folder type in dataset folders response: " + string(folder.Type) + ". Root level folders are expected to be of type 'folder'.")
		}

		folderModel.Type = types.StringValue(string(folder.Type))
		if err := folderModel.updateState(&folder); err != nil {
			return err
		}
		folderModel.Uuid = types.StringValue(folder.Uuid.String())

		folders = append(folders, folderModel)
	}
	model.Folders = folders

	return nil
}

func (model *datasetFolderModel) updateState(d *client.Folder) error {
	model.Name = types.StringValue(d.Name)
	if d.Description.IsSpecified() && d.Description.MustGet() != "" {
		model.Description = types.StringValue(d.Description.MustGet())
	}

	if d.Type != client.FolderTypeFolder {
		return errors.New("unexpected folder type in dataset folders response: " + string(d.Type) + ". Nested folders are not supported.")
	}
	model.Type = types.StringValue(string(d.Type))
	model.Uuid = types.StringValue(d.Uuid.String())

	var children []datasetFolderChildModel
	if !d.Children.IsNull() && len(d.Children.MustGet()) > 0 {
		for _, child := range d.Children.MustGet() {
			childModel := datasetFolderChildModel{
				Name: types.StringValue(child.Name),
				Type: types.StringValue(string(child.Type)),
			}
			if child.Description.IsSpecified() && child.Description.MustGet() != "" {
				childModel.Description = types.StringValue(child.Description.MustGet())
			}

			u := child.Uuid.String()
			if u == "" || u == "00000000-0000-0000-0000-000000000000" {
				return fmt.Errorf("missing uuid for child %q in folders response", child.Name)
			} else {
				childModel.Uuid = types.StringValue(child.Uuid.String())
			}

			children = append(children, childModel)
		}

		model.Children = children
	}

	return nil
}

func findColumnByName(columns []client.DatasetRestApiGetTableColumn, name string) *client.DatasetRestApiGetTableColumn {
	for _, column := range columns {
		if column.ColumnName == name {
			return &column
		}
	}
	return nil
}

func findMetricByName(metrics []client.DatasetRestApiGetSqlMetric, name string) *client.DatasetRestApiGetSqlMetric {
	for _, metric := range metrics {
		if metric.MetricName == name {
			return &metric
		}
	}
	return nil
}

func (model *datasetFolderBaseModel) resolveColumns(d *client.DatasetRestApiGet) {

	for i, folder := range model.Folders {
		if folder.Type.ValueString() == string(client.FolderTypeFolder) {
			model.Folders[i].resolveColumns(d)
		}
	}
}

func (model *datasetFolderModel) resolveColumns(d *client.DatasetRestApiGet) {
	for i, child := range model.Children {
		if child.Type.ValueString() == string(client.FolderTypeColumn) {
			column := findColumnByName(d.Columns, child.Name.ValueString())
			if column != nil && column.Uuid.IsSpecified() {
				model.Children[i].Uuid = types.StringValue(column.Uuid.MustGet().String())
			}
		} else if child.Type.ValueString() == string(client.FolderTypeMetric) {
			metric := findMetricByName(d.Metrics, child.Name.ValueString())
			if metric != nil && metric.Uuid.IsSpecified() {
				model.Children[i].Uuid = types.StringValue(metric.Uuid.MustGet().String())
			}
		}
	}
}

func (model *datasetFolderBaseModel) toFolders() ([]client.Folder, error) {
	var folders []client.Folder
	for _, folderModel := range model.Folders {
		folder := client.Folder{
			Name:     folderModel.Name.ValueString(),
			Type:     client.FolderType(folderModel.Type.ValueString()),
			Children: nullable.NewNullableWithValue([]client.Folder{}),
		}
		if folderModel.Uuid.IsNull() || folderModel.Uuid.ValueString() == "" || folderModel.Uuid.ValueString() == "00000000-0000-0000-0000-000000000000" {
			folder.Uuid = uuid.New()
		} else {
			uuid, err := uuid.Parse(folderModel.Uuid.ValueString())
			if err != nil {
				return nil, err
			}
			folder.Uuid = uuid
		}

		if !folderModel.Description.IsNull() && folderModel.Description.ValueString() != "" {
			folder.Description = nullable.NewNullableWithValue(folderModel.Description.ValueString())
		}
		children, err := folderModel.toFolders()
		if err != nil {
			return nil, err
		}
		if len(children) > 0 {
			folder.Children = nullable.NewNullableWithValue(children)
		}
		folders = append(folders, folder)
	}

	return folders, nil
}

func (model *datasetFolderModel) toFolders() ([]client.Folder, error) {
	var folders []client.Folder
	for _, child := range model.Children {
		folder := client.Folder{
			Name:     child.Name.ValueString(),
			Type:     client.FolderType(child.Type.ValueString()),
			Children: nullable.NewNullableWithValue([]client.Folder{}),
		}
		if child.Uuid.IsNull() || child.Uuid.ValueString() == "" || child.Uuid.ValueString() == "00000000-0000-0000-0000-000000000000" {
			folder.Uuid = uuid.New()
		} else {
			uuid, err := uuid.Parse(child.Uuid.ValueString())
			if err != nil {
				return nil, err
			}
			folder.Uuid = uuid
		}

		if !child.Description.IsNull() && child.Description.ValueString() != "" {
			folder.Description = nullable.NewNullableWithValue(child.Description.ValueString())
		}
		folders = append(folders, folder)
	}

	return folders, nil
}

func mapToFolders(m interface{}) ([]client.Folder, error) {
	var folders []client.Folder

	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &folders); err != nil {
		return nil, err
	}

	return folders, nil
}
