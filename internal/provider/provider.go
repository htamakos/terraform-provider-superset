// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/htamakos/terraform-provider-superset/internal/client"
)

var _ provider.Provider = &SupersetProvider{}

type SupersetProvider struct {
	version string
}

type SupersetProviderModel struct {
	ServerBaseUrl types.String `tfsdk:"server_base_url"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	PageSize      types.Int64  `tfsdk:"page_size"`
}

func (p *SupersetProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "superset"
	resp.Version = p.version
}

func (p *SupersetProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server_base_url": schema.StringAttribute{
				MarkdownDescription: "The base URL of the Superset server.",
				Optional:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "The username for Superset authentication.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for Superset authentication.",
				Sensitive:           true,
				Optional:            true,
			},
			"page_size": schema.Int64Attribute{
				MarkdownDescription: "The number of items to retrieve per page when paginating through API results.",
				Optional:            true,
			},
		},
	}
}

func (p *SupersetProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Check environment variables
	serverBaseUrl := os.Getenv("SUPERSET_SERVER_BASE_URL")
	username := os.Getenv("SUPERSET_USERNAME")
	password := os.Getenv("SUPERSET_PASSWORD")
	pageSize := client.DefaultPageSize

	var data SupersetProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if data.ServerBaseUrl.ValueString() != "" {
		serverBaseUrl = data.ServerBaseUrl.ValueString()
	}

	if !data.Username.IsNull() {
		username = data.Username.ValueString()
	}

	if !data.Password.IsNull() {
		password = data.Password.ValueString()
	}

	if !data.PageSize.IsNull() {
		pageSize = int(data.PageSize.ValueInt64())
	}

	if serverBaseUrl == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("server_base_url"),
			"Missing Configuration",
			"The provider cannot create the client as there is no value set for the Superset server base URL. "+
				"Please set the server_base_url attribute in the provider configuration or the SUPERSET_SERVER_BASE_URL environment variable. ",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Configuration",
			"The provider cannot create the client as there is no value set for the Superset username. "+
				"Please set the username attribute in the provider configuration or the SUPERSET_USERNAME environment variable. ",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Configuration",
			"The provider cannot create the client as there is no value set for the Superset password. "+
				"Please set the password attribute in the provider configuration or the SUPERSET_PASSWORD environment variable. ",
		)
	}

	if pageSize < 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("page_size"),
			"Invalid Configuration",
			"The provider cannot create the client as the page_size cannot be negative. "+
				"Please set the page_size attribute in the provider configuration to a non-negative value. ",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.NewClientWrapper(ctx,
		serverBaseUrl,
		client.ClientCredentials{Username: username, Password: password},
		client.WithPageSize(pageSize),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Superset Client",
			"An unexpected error was encountered trying to create the Superset client. "+
				"Please check the server base URL and credentials are correct and try again. "+
				"Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured Superset client", map[string]interface{}{
		"server_base_url": serverBaseUrl,
		"username":        username,
		"page_size":       pageSize,
	})
}

func (p *SupersetProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
        NewUserResource,
        NewRoleResource,
        NewRolePermissionsResource,
        NewGroupResource,
        NewGroupRoleBindingResource,
	}
}

func (p *SupersetProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SupersetProvider{
			version: version,
		}
	}
}
