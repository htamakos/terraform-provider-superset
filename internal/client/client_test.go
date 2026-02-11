// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
)

var (
	testServerBaseUrl  = "http://localhost:8088"
	testServerUser     = "admin"
	testServerPassword = "admin"
)

func skipIfNoClientTest(t *testing.T) {
	if os.Getenv("EXECUTE_CLIENT_TEST") != "1" {
		t.Skip("Skipping client tests. Set EXECUTE_CLIENT_TEST=1 to run them.")
	}
}

func TestAuthenticate(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWithResponses(testServerBaseUrl)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	body := PostApiV1SecurityLoginJSONRequestBody{
		Username: testServerUser,
		Password: testServerPassword,
		Provider: defaultLoginProvider,
	}

	token, err := authenticate(ctx, client, body)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	if token == "" {
		t.Fatalf("empty access token")
	}
}

// TestUserApis tests user-related APIs.
func TestUserApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Create a new user
	u, err := client.CreateUser(ctx, SupersetUserApiPost{
		Username: "testuser3",
		Email:    "test-user3@example.com",
		Roles:    []int{1},
		Groups:   []int{},
	})

	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user, err := client.FindUser(ctx, u.Username)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	_, err = client.FindUser(ctx, "nonexistentuser")
	if err == nil {
		t.Fatalf("expected error when finding nonexistent user, got nil")
	}

	if !strings.HasPrefix(err.Error(), "user not found") {
		t.Fatalf("unexpected error message: %v", err)
	}

	g, err := client.CreateGroup(ctx, SupersetGroupApiPost{
		Name: "testgroupforuser",
	})
	if err != nil {
		t.Fatalf("failed to create group for user: %v", err)
	}

	_, err = client.UpdateUser(ctx, user.Id, SupersetUserApiPut{
		FirstName: "Test",
		Password:  "newpassword123",
		Groups:    []int{g.Id},
	})

	if err != nil {
		t.Fatalf("failed to update user: %v", err)
	}

	_, err = client.GetUser(ctx, user.Id)
	if err != nil {
		t.Fatalf("failed to get user after update: %v", err)
	}

	_, err = client.UpdateUser(ctx, user.Id, SupersetUserApiPut{
		Groups: []int{},
	})
	if err != nil {
		t.Fatalf("failed to update user to remove groups: %v", err)
	}

	users, err := client.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}

	if len(users) == 0 {
		t.Fatalf("expected at least one user, got zero")
	}

	err = client.DeleteUser(ctx, user.Id)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Clean up: delete the created group
	err = client.DeleteGroup(ctx, g.Id)
	if err != nil {
		t.Fatalf("failed to delete group: %v", err)
	}
}

func TestRoleApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	r, err := client.CreateRole(ctx, SupersetRoleApiPost{
		Name: "testrole",
	})

	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	role, err := client.FindRole(ctx, r.Name)
	if err != nil {
		t.Fatalf("failed to find role: %v", err)
	}

	_, err = client.UpdateRole(ctx, role.Id, SupersetRoleApiPut{
		Name: "updatedtestrole",
	})
	if err != nil {
		t.Fatalf("failed to update role: %v", err)
	}

	roles, err := client.ListRoles(ctx)
	if err != nil {
		t.Fatalf("failed to list roles: %v", err)
	}

	if len(roles) == 0 {
		t.Fatalf("expected at least one role, got zero")
	}

	// Clean up: delete the created role
	err = client.DeleteRole(ctx, role.Id)
	if err != nil {
		t.Fatalf("failed to delete role: %v", err)
	}
}

func TestGroupApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	g, err := client.CreateGroup(ctx, SupersetGroupApiPost{
		Name: "testgroup",
	})

	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	group, err := client.FindGroup(ctx, g.Name)
	if err != nil {
		t.Fatalf("failed to find group: %v", err)
	}

	roles, err := client.ListRoles(ctx)
	if err != nil {
		t.Fatalf("failed to list roles: %v", err)
	}

	roleIds := make([]int, len(roles))
	for i, role := range roles {
		roleIds[i] = role.Id
	}

	users, err := client.ListUsers(ctx)
	if err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	userIds := make([]int, len(users))
	for i, user := range users {
		userIds[i] = user.Id
	}

	_, err = client.UpdateGroup(ctx, group.Id, SupersetGroupApiPut{
		Name:  "updatedtestgroup",
		Roles: roleIds,
		Users: userIds,
	})

	if err != nil {
		t.Fatalf("failed to update group: %v", err)
	}

	groups, err := client.ListGroups(ctx)
	if err != nil {
		t.Fatalf("failed to list groups: %v", err)
	}

	if len(groups) == 0 {
		t.Fatalf("expected at least one group, got zero")
	}

	// Clean up: delete the created group
	err = client.DeleteGroup(ctx, group.Id)
	if err != nil {
		t.Fatalf("failed to delete group: %v", err)
	}
}

func TestRolePermissionsApi(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ListRolePermissions(ctx, 1)
	if err != nil {
		t.Fatalf("failed to list role permissions: %v", err)
	}
}

func TestResourcePermissionsApi(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ListPermissions(ctx)
	if err != nil {
		t.Fatalf("failed to list resource permissions: %v", err)
	}
}

func TestDatabaseApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.CreateDatabase(ctx, DatabaseRestApiPost{
		AllowCtas:      false,
		AllowDml:       false,
		DatabaseName:   "test_database",
		ExposeInSqllab: true,
		SqlalchemyUri:  "postgresql+psycopg2://superset:superset@db:5432/superset",
	})
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	err = client.ExecuteTestDatabaseConnection(ctx, DatabaseTestConnectionSchema{
		SqlalchemyUri: "postgresql+psycopg2://superset:superset@db:5432/superset",
	})

	if err != nil {
		t.Fatalf("failed to test database connection: %v", err)
	}

	database, err := client.FindDatabase(ctx, "test_database")
	if err != nil {
		t.Fatalf("failed to find database: %v", err)
	}

	err = client.UpdateDatabase(ctx, database.Id, DatabaseRestApiPut{
		AllowCtas:       true,
		ImpersonateUser: false,
	})

	if err != nil {
		t.Fatalf("failed to update database: %v", err)
	}

	databases, err := client.ListDatabases(ctx)
	if err != nil {
		t.Fatalf("failed to list databases: %v", err)
	}
	if len(databases) == 0 {
		t.Fatalf("expected at least one database, got zero")
	}

	err = client.DeleteDatabase(ctx, database.Id)
	if err != nil {
		t.Fatalf("failed to delete database: %v", err)
	}
}

func TestTagApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tag, err := client.CreateTag(ctx, TagRestApiPost{
		Name:        "testtag",
		Description: nullable.NewNullableWithValue("This is a test tag"),
	})
	if err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	if tag.Name != "testtag" {
		t.Fatalf("unexpected tag name: %v", tag.Name)
	}

	foundTag, err := client.FindTag(ctx, "testtag")
	if err != nil {
		t.Fatalf("failed to find tag: %v", err)
	}

	if foundTag.Id != tag.Id {
		t.Fatalf("found tag ID does not match created tag ID")
	}

	ltags, err := client.ListTags(ctx)
	if err != nil {
		t.Fatalf("failed to list tags: %v", err)
	}

	if len(ltags) == 0 {
		t.Fatalf("expected at least one tag, got zero")
	}

	updatedTag, err := client.UpdateTag(ctx, tag.Id, TagRestApiPut{
		Name:        "testtag",
		Description: nullable.NewNullableWithValue("Updated description"),
	})
	if err != nil {
		t.Fatalf("failed to update tag: %v", err)
	}

	if updatedTag.Description.IsNull() || updatedTag.Description.MustGet() != "Updated description" {
		t.Fatalf("unexpected tag description after update: %v", updatedTag.Description)
	}

	err = client.DeleteTag(ctx, tag.Id)
	if err != nil {
		t.Fatalf("failed to delete tag: %v", err)
	}
}

func TestDatasetApis(t *testing.T) {
	skipIfNoClientTest(t)

	ctx := context.Background()
	client, err := NewClientWrapper(ctx, testServerBaseUrl, ClientCredentials{Username: testServerUser, Password: testServerPassword})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	database, err := client.CreateDatabase(ctx, DatabaseRestApiPost{
		AllowCtas:      false,
		AllowDml:       false,
		DatabaseName:   "test_database_for_dataset",
		ExposeInSqllab: true,
		SqlalchemyUri:  "postgresql+psycopg2://superset:superset@db:5432/superset",
	})
	if err != nil {
		t.Fatalf("failed to create database for dataset: %v", err)
	}

	dataset, err := client.CreateDataset(ctx, DatasetRestApiPost{
		Database:  database.Id,
		Schema:    nullable.NewNullableWithValue("information_schema"),
		TableName: "tables",
	})
	if err != nil {
		t.Fatalf("failed to create dataset: %v", err)
	}

	if dataset.TableName != "tables" {
		t.Fatalf("unexpected dataset table name: %v", dataset.TableName)
	}

	datasets, err := client.ListDatasets(ctx)
	if err != nil {
		t.Fatalf("failed to list datasets: %v", err)
	}

	if len(datasets) == 0 {
		t.Fatalf("expected at least one dataset, got zero")
	}

	datasetFound, err := client.FindDataset(ctx, "tables")
	if err != nil {
		t.Fatalf("failed to find dataset: %v", err)
	}

	if datasetFound.Id != dataset.Id {
		t.Fatalf("found dataset ID does not match created dataset ID")
	}
	if datasetFound.TableName != dataset.TableName {
		t.Fatalf("found dataset TableName does not match created dataset TableName")
	}

	datasetUpdated, err := client.UpdateDataset(ctx, dataset.Id, DatasetRestApiPut{
		NormalizeColumns:     nullable.NewNullableWithValue(true),
		AlwaysFilterMainDttm: true,
	})
	if err != nil {
		t.Fatalf("failed to update dataset: %v", err)
	}

	if datasetUpdated.NormalizeColumns.IsNull() || !datasetUpdated.NormalizeColumns.MustGet() {
		t.Fatalf("unexpected dataset NormalizeColumns after update: %v", datasetUpdated.NormalizeColumns)
	}
	if datasetUpdated.AlwaysFilterMainDttm.IsNull() || !datasetUpdated.AlwaysFilterMainDttm.MustGet() {
		t.Fatalf("unexpected dataset AlwaysFilterMainDttm after update: %v", datasetUpdated.AlwaysFilterMainDttm)
	}

	err = client.DeleteDataset(ctx, dataset.Id)
	if err != nil {
		t.Fatalf("failed to delete dataset: %v", err)
	}

	err = client.DeleteDatabase(ctx, database.Id)
	if err != nil {
		t.Fatalf("failed to delete database: %v", err)
	}
}
