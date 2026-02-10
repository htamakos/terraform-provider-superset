// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const defaultLoginProviderName string = "db"
const DefaultPageSize int = 4096

var defaultLoginProvider = PostApiV1SecurityLoginJSONBodyProvider(defaultLoginProviderName)

// ClientWrapper wraps the generated ClientWithResponses to add authentication handling.
type ClientWrapper struct {
	*ClientWithResponses
	pageSize int
}

// accessToken represents an authentication access token.
type accessToken string

// ClientOptions holds options for creating a ClientWrapper.
type ClientOptions struct {
	PageSize int
}

// ClientCredentials holds the username and password for authentication.
type ClientCredentials struct {
	Username string
	Password string
}

type clientOptionFn func(*ClientOptions)

func WithPageSize(pageSize int) clientOptionFn {
	return func(opts *ClientOptions) {
		opts.PageSize = pageSize
	}
}

// NotFoundError represents 404 from API.
type NotFoundError struct {
	Resource string
	ID       any
}

func (e *NotFoundError) Error() string {
	if e.ID != nil {
		return fmt.Sprintf("%s not found (id=%v)", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// IsNotFound checks if the error is a NotFoundError.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

// NewClientWrapper creates a new ClientWrapper with authentication.
func NewClientWrapper(ctx context.Context, serverBaseUrl string, credentials ClientCredentials, optionFns ...clientOptionFn) (*ClientWrapper, error) {
	// Create initial client without authentication to perform login
	client, err := NewClientWithResponses(serverBaseUrl)
	if err != nil {
		return nil, err
	}

	body := PostApiV1SecurityLoginJSONRequestBody{
		Username: credentials.Username,
		Password: credentials.Password,
		Provider: defaultLoginProvider,
	}

	accessToken, err := authenticate(ctx, client, body)
	if err != nil {
		return nil, err
	}

	client, err = NewClientWithResponses(serverBaseUrl, WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		return nil
	}))
	if err != nil {
		return nil, err
	}

	clientOptions := &ClientOptions{
		PageSize: DefaultPageSize,
	}
	for _, fn := range optionFns {
		fn(clientOptions)
	}

	cw := &ClientWrapper{
		client,
		clientOptions.PageSize,
	}

	return cw, nil
}

// authenticate performs authentication and returns the access token.
func authenticate(ctx context.Context, client *ClientWithResponses, body PostApiV1SecurityLoginJSONRequestBody) (accessToken, error) {
	res, err := client.PostApiV1SecurityLoginWithResponse(ctx, body)
	if err != nil {
		return "", err
	}

	if res.StatusCode() != http.StatusOK {
		errMsg := string(res.Body)

		return "", fmt.Errorf("authentication failed with status code: %d, message: %s", res.StatusCode(), errMsg)
	}

	return accessToken(res.JSON200.AccessToken), nil
}

func (cw *ClientWrapper) createCsrfTokenRequestEditor() (RequestEditorFn, error) {
	csrfToken, cookies, err := cw.GetCsrfTokenAndCookies(context.Background())
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Add("x-csrftoken", csrfToken)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		return nil
	}, nil
}

func (cw *ClientWrapper) GetCsrfTokenAndCookies(ctx context.Context) (string, []*http.Cookie, error) {
	res, err := cw.GetApiV1SecurityCsrfTokenWithResponse(ctx)
	if err != nil {
		return "", nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return "", nil, fmt.Errorf("failed to get CSRF token, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, res.HTTPResponse.Cookies(), nil
}

// ListUsers retrieves the list of users.
func (cw *ClientWrapper) ListUsers(ctx context.Context) ([]SupersetUserApiGetList, error) {
	pageNumber := 0
	var allUsers []SupersetUserApiGetList
	for {
		users, err := cw._ListUsers(ctx, pageNumber)
		if err != nil {
			return nil, err
		}
		allUsers = append(allUsers, users...)
		if len(users) < cw.pageSize {
			break
		}
		pageNumber++
	}
	return allUsers, nil
}

func (cw *ClientWrapper) _ListUsers(ctx context.Context, pageNumber int) ([]SupersetUserApiGetList, error) {
	res, err := cw.GetApiV1SecurityUsersWithResponse(ctx, &GetApiV1SecurityUsersParams{
		Q: GetListSchema{
			Page:     pageNumber,
			PageSize: cw.pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get users, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, nil
}

// CreateUser creates a new user with the given user data.
func (cw *ClientWrapper) CreateUser(ctx context.Context, user SupersetUserApiPost) (*SupersetUserApiGet, error) {
	res, err := cw.PostApiV1SecurityUsers(ctx, user)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to create user, status code: %d, body: %s", res.StatusCode, string(msg))
	}

	userRes, err := ParsePostApiV1SecurityUsersResponse(res)
	if err != nil {
		return nil, err
	}

	cwUser, err := cw.GetApiV1SecurityUsersPkWithResponse(ctx, userRes.JSON201.Id, nil)
	if err != nil {
		return nil, err
	}

	if cwUser.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get created user, status code: %d, body: %s", cwUser.StatusCode(), string(cwUser.Body))
	}

	return &cwUser.JSON200.Result, nil
}

// GetUser retrieves the user with the given userID.
func (cw *ClientWrapper) GetUser(ctx context.Context, userID int) (*SupersetUserApiGet, error) {
	res, err := cw.GetApiV1SecurityUsersPkWithResponse(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get user, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "User", ID: userID}
	}

	return &res.JSON200.Result, nil
}

// FindUser finds a user by username.
func (cw *ClientWrapper) FindUser(ctx context.Context, userName string) (*SupersetUserApiGetList, error) {
	var v GetListSchema_Filters_Value
	err := v.FromGetListSchemaFiltersValue1(userName)
	if err != nil {
		return nil, err
	}

	res, err := cw.GetApiV1SecurityUsersWithResponse(ctx, &GetApiV1SecurityUsersParams{
		Q: GetListSchema{
			Filters: []struct {
				Col   string                      `json:"col"`
				Opr   string                      `json:"opr"`
				Value GetListSchema_Filters_Value `json:"value"`
			}{
				{Col: "username", Opr: "eq", Value: v},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "User", ID: userName}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to find user, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, &NotFoundError{Resource: "User", ID: userName}
	}

	return &res.JSON200.Result[0], nil
}

// DeleteUser deletes the user with the given userID.
func (cw *ClientWrapper) DeleteUser(ctx context.Context, userID int) error {
	res, err := cw.DeleteApiV1SecurityUsersPk(ctx, userID)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to delete user, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

// UpdateUser updates the user with the given userID using the provided user data.
func (cw *ClientWrapper) UpdateUser(ctx context.Context, userID int, user SupersetUserApiPut) (*SupersetUserApiGet, error) {
	fmt.Printf("Updating user ID %d with data: %+v\n", userID, user)
	res, err := cw.PutApiV1SecurityUsersPk(ctx, userID, user)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to update user, status code: %d, body: %s", res.StatusCode, string(msg))
	}

	u, err := cw.GetApiV1SecurityUsersPkWithResponse(ctx, userID, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get updated user: %w", err)
	}

	if u.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get user, status code: %d, body: %s", u.StatusCode(), string(u.Body))
	}

	return &u.JSON200.Result, nil
}

// ListRoles retrieves the list of roles.
func (cw *ClientWrapper) ListRoles(ctx context.Context) ([]SupersetRoleApiGetList, error) {
	pageNumber := 0
	var allRoles []SupersetRoleApiGetList
	for {
		roles, err := cw._ListRoles(ctx, pageNumber)
		if err != nil {
			return nil, err
		}
		allRoles = append(allRoles, roles...)
		if len(roles) < cw.pageSize {
			break
		}
		pageNumber++
	}
	return allRoles, nil
}

func (cw *ClientWrapper) _ListRoles(ctx context.Context, pageNumber int) ([]SupersetRoleApiGetList, error) {
	res, err := cw.GetApiV1SecurityRolesWithResponse(ctx, &GetApiV1SecurityRolesParams{
		Q: GetListSchema{
			Page:     pageNumber,
			PageSize: cw.pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get roles, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, nil
}

// FindRole finds a role by role name.
func (cw *ClientWrapper) FindRole(ctx context.Context, roleName string) (*SupersetRoleApiGetList, error) {
	var v GetListSchema_Filters_Value
	err := v.FromGetListSchemaFiltersValue1(roleName)
	if err != nil {
		return nil, err
	}

	res, err := cw.GetApiV1SecurityRolesWithResponse(ctx, &GetApiV1SecurityRolesParams{
		Q: GetListSchema{
			Filters: []struct {
				Col   string                      `json:"col"`
				Opr   string                      `json:"opr"`
				Value GetListSchema_Filters_Value `json:"value"`
			}{
				{Col: "name", Opr: "eq", Value: v},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "Role", ID: roleName}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to find role, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, &NotFoundError{Resource: "Role", ID: roleName}
	}

	return &res.JSON200.Result[0], nil
}

// CreateRole creates a new role with the given role data.
func (cw *ClientWrapper) CreateRole(ctx context.Context, role SupersetRoleApiPost) (*SupersetRoleApiGet, error) {
	res, err := cw.PostApiV1SecurityRoles(ctx, role)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to create role, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	createdRoleRes, err := ParsePostApiV1SecurityRolesResponse(res)
	if err != nil {
		return nil, err
	}

	cwRole, err := cw.GetApiV1SecurityRolesPkWithResponse(ctx, createdRoleRes.JSON201.Id, nil)
	if err != nil {
		return nil, err
	}
	if cwRole.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get created role, status code: %d, body: %s", cwRole.StatusCode(), string(cwRole.Body))
	}

	return &cwRole.JSON200.Result, nil
}

// GetRole retrieves the role with the given roleID.
func (cw *ClientWrapper) GetRole(ctx context.Context, roleID int) (*SupersetRoleApiGet, error) {
	res, err := cw.GetApiV1SecurityRolesPkWithResponse(ctx, roleID, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "Role", ID: roleID}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get role, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return &res.JSON200.Result, nil
}

// DeleteRole deletes the role with the given roleID.
func (cw *ClientWrapper) DeleteRole(ctx context.Context, roleID int) error {
	res, err := cw.DeleteApiV1SecurityRolesPk(ctx, roleID)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to delete role, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

// UpdateRole updates the role with the given roleID using the provided role data.
func (cw *ClientWrapper) UpdateRole(ctx context.Context, roleID int, role SupersetRoleApiPut) (*SupersetRoleApiGet, error) {
	res, err := cw.PutApiV1SecurityRolesPk(ctx, roleID, role)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to update role, status code: %d, body: %s", res.StatusCode, string(msg))
	}

	roleRes, err := cw.GetApiV1SecurityRolesPkWithResponse(ctx, roleID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated role: %w", err)
	}

	if roleRes.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get role, status code: %d, body: %s", roleRes.StatusCode(), string(roleRes.Body))
	}

	return &roleRes.JSON200.Result, nil
}

// Groups
// ListGroups retrieves the list of groups.
type SupersetGroupApiGetList = GroupApiGetList

func (cw *ClientWrapper) ListGroups(ctx context.Context) ([]SupersetGroupApiGetList, error) {
	pageNumber := 0
	var allGroups []SupersetGroupApiGetList
	for {
		groups, err := cw._ListGroups(ctx, pageNumber)
		if err != nil {
			return nil, err
		}
		allGroups = append(allGroups, groups...)
		if len(groups) < cw.pageSize {
			break
		}
		pageNumber++
	}
	return allGroups, nil
}

func (cw *ClientWrapper) _ListGroups(ctx context.Context, pageNumber int) ([]SupersetGroupApiGetList, error) {
	res, err := cw.GetApiV1SecurityGroupsWithResponse(ctx, &GetApiV1SecurityGroupsParams{
		Q: GetListSchema{
			Page:     pageNumber,
			PageSize: cw.pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get groups, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, nil
}

// GetGroup retrieves the group with the given groupID.
func (cw *ClientWrapper) GetGroup(ctx context.Context, groupID int) (*SupersetGroupApiGet, error) {
	res, err := cw.GetApiV1SecurityGroupsPkWithResponse(ctx, groupID, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "Group", ID: groupID}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get group, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}
	return &res.JSON200.Result, nil
}

// FindGroup finds a group by group name.
func (cw *ClientWrapper) FindGroup(ctx context.Context, groupName string) (*SupersetGroupApiGetList, error) {
	var v GetListSchema_Filters_Value
	err := v.FromGetListSchemaFiltersValue1(groupName)
	if err != nil {
		return nil, err
	}

	res, err := cw.GetApiV1SecurityGroupsWithResponse(ctx, &GetApiV1SecurityGroupsParams{
		Q: GetListSchema{
			Filters: []struct {
				Col   string                      `json:"col"`
				Opr   string                      `json:"opr"`
				Value GetListSchema_Filters_Value `json:"value"`
			}{
				{Col: "name", Opr: "eq", Value: v},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "Group", ID: groupName}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to find group, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, &NotFoundError{Resource: "Group", ID: groupName}
	}

	return &res.JSON200.Result[0], nil
}

type SupersetGroupApiPost = PostApiV1SecurityGroupsJSONRequestBody
type SupersetGroupApiGet = GroupApiGet

// CreateGroup creates a new group with the given group data.
func (cw *ClientWrapper) CreateGroup(ctx context.Context, group SupersetGroupApiPost) (*SupersetGroupApiGet, error) {
	res, err := cw.PostApiV1SecurityGroups(ctx, group)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to create group, status code: %d, body: %s", res.StatusCode, string(msg))
	}

	createdGroupRes, err := ParsePostApiV1SecurityGroupsResponse(res)
	if err != nil {
		return nil, err
	}

	cwGroup, err := cw.GetApiV1SecurityGroupsPkWithResponse(ctx, createdGroupRes.JSON201.Id, nil)
	if err != nil {
		return nil, err
	}

	if cwGroup.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get created group, status code: %d, body: %s", cwGroup.StatusCode(), string(cwGroup.Body))
	}

	return &cwGroup.JSON200.Result, nil
}

// DeleteGroup deletes the group with the given groupID.
func (cw *ClientWrapper) DeleteGroup(ctx context.Context, groupID int) error {
	res, err := cw.DeleteApiV1SecurityGroupsPk(ctx, groupID)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to delete group, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

type SupersetGroupApiPut = PutApiV1SecurityGroupsPkJSONRequestBody

// UpdateGroup updates the group with the given groupID using the provided group data.
func (cw *ClientWrapper) UpdateGroup(ctx context.Context, groupID int, group SupersetGroupApiPut) (*SupersetGroupApiGet, error) {
	res, err := cw.PutApiV1SecurityGroupsPk(ctx, groupID, group)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to update group, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	groupRes, err := cw.GetApiV1SecurityGroupsPkWithResponse(ctx, groupID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated group: %w", err)
	}

	if groupRes.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get group, status code: %d, body: %s", groupRes.StatusCode(), string(groupRes.Body))
	}

	return &groupRes.JSON200.Result, nil
}

type SupersetPermissionApiGetList = PermissionViewMenuApiGetList

// ListPermissions retrieves the list of permissions.
func (cw *ClientWrapper) ListPermissions(ctx context.Context) ([]SupersetPermissionApiGetList, error) {
	pageNumber := 0
	var allPermissions []SupersetPermissionApiGetList
	for {
		permissions, err := cw._ListPermissions(ctx, pageNumber)
		if err != nil {
			return nil, err
		}

		allPermissions = append(allPermissions, permissions...)

		if len(permissions) == 0 {
			break
		}
		pageNumber++
	}
	return allPermissions, nil
}

func (cw *ClientWrapper) _ListPermissions(ctx context.Context, pageNumber int) ([]SupersetPermissionApiGetList, error) {
	res, err := cw.GetApiV1SecurityPermissionsResourcesWithResponse(ctx, &GetApiV1SecurityPermissionsResourcesParams{
		Q: GetListSchema{
			Page:     pageNumber,
			PageSize: cw.pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get permissions, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, nil
}

// Role Permissions

type SupersetRolePermissionApiGetList = RolePermissionListSchema

// ListRolePermissions retrieves the list of permissions for a given role ID.
func (cw *ClientWrapper) ListRolePermissions(ctx context.Context, roleId int) ([]SupersetRolePermissionApiGetList, error) {
	res, err := cw.GetApiV1SecurityRolesRoleIdPermissionsWithResponse(ctx, roleId)
	if err != nil {
		return nil, err
	}

	if res.StatusCode() == http.StatusNotFound {
		return nil, &NotFoundError{Resource: "Role", ID: roleId}
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get permissions, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, &NotFoundError{Resource: "Role Permissions", ID: roleId}
	}

	return res.JSON200.Result, nil
}

// AssignPermissionsToRole assigns the given permission IDs to the specified role ID.
func (cw *ClientWrapper) AssignPermissionsToRole(ctx context.Context, roleId int, permissionIds []int) error {
	res, err := cw.PostApiV1SecurityRolesRoleIdPermissions(ctx, roleId, RolePermissionPostSchema{
		PermissionViewMenuIds: permissionIds,
	})

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to add role permissions, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

// AssignRolesToGroup assigns the given role IDs to the specified group ID.
func (cw *ClientWrapper) AssignRolesToGroup(ctx context.Context, groupId int, roleIds []int) error {
	body := SupersetGroupApiPut{
		Roles: roleIds,
	}

	_, err := cw.UpdateGroup(ctx, groupId, body)
	if err != nil {
		return err
	}

	return nil
}

// AssignUsersToGroup assigns the given user IDs to the specified group ID.
func (cw *ClientWrapper) AssignUsersToGroup(ctx context.Context, groupId int, userIds []int) error {
	body := SupersetGroupApiPut{
		Users: userIds,
	}

	_, err := cw.UpdateGroup(ctx, groupId, body)
	if err != nil {
		return err
	}

	return nil
}

// AssignUsersToRole assigns the given user IDs to the specified role ID.
func (cw *ClientWrapper) AssignUsersToRole(ctx context.Context, roleId int, userIds []int) error {
	res, err := cw.PutApiV1SecurityRolesRoleIdUsers(ctx, roleId, RoleUserPutSchema{
		UserIds: userIds,
	})

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to add role users, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

type SupersetDatabaseApiGetList = DatabaseRestApiGetList

func (cw *ClientWrapper) ListDatabases(ctx context.Context) ([]SupersetDatabaseApiGetList, error) {
	pageNumber := 0
	var allDatabases []SupersetDatabaseApiGetList
	for {
		databases, err := cw._ListDatabases(ctx, pageNumber)
		if err != nil {
			return nil, err
		}
		allDatabases = append(allDatabases, databases...)
		if len(databases) < cw.pageSize {
			break
		}
		pageNumber++
	}
	return allDatabases, nil
}

func (cw *ClientWrapper) _ListDatabases(ctx context.Context, pageNumber int) ([]SupersetDatabaseApiGetList, error) {
	res, err := cw.GetApiV1DatabaseWithResponse(ctx, &GetApiV1DatabaseParams{
		Q: GetListSchema{
			Page:     pageNumber,
			PageSize: cw.pageSize,
		},
	})
	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to get databases, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	return res.JSON200.Result, nil
}

type SupersetDatabaseApiGet = SupersetDatabaseApiGetList

// FindDatabase finds a database by database name.
func (cw *ClientWrapper) FindDatabase(ctx context.Context, databaseName string) (*SupersetDatabaseApiGetList, error) {
	var v GetListSchema_Filters_Value
	err := v.FromGetListSchemaFiltersValue1(databaseName)
	if err != nil {
		return nil, err
	}

	res, err := cw.GetApiV1DatabaseWithResponse(ctx, &GetApiV1DatabaseParams{
		Q: GetListSchema{
			Filters: []struct {
				Col   string                      `json:"col"`
				Opr   string                      `json:"opr"`
				Value GetListSchema_Filters_Value `json:"value"`
			}{
				{Col: "database_name", Opr: "eq", Value: v},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to find database, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, fmt.Errorf("database not found: %s", databaseName)
	}

	return &res.JSON200.Result[0], nil
}

// CreateDatabase creates a new database with the given database data.
type SupersetDatabaseApiPost = DatabaseRestApiPost

func (cw *ClientWrapper) CreateDatabase(ctx context.Context, database SupersetDatabaseApiPost) (*DatabaseRestApiGetList, error) {
	reqEditor, err := cw.createCsrfTokenRequestEditor()
	if err != nil {
		return nil, err
	}

	res, err := cw.PostApiV1Database(ctx, database, reqEditor)

	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusCreated {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		return nil, fmt.Errorf("failed to create database, status code: %d, body: %s", res.StatusCode, string(msg))
	}

	databaseRes, err := cw.FindDatabase(ctx, database.DatabaseName)
	if err != nil {
		return nil, err
	}

	return databaseRes, nil
}

// GetDatabase retrieves the database with the given databaseID.
func (cw *ClientWrapper) GetDatabase(ctx context.Context, databaseID int) (*DatabaseRestApiGetList, error) {
	var v GetListSchema_Filters_Value
	err := v.FromGetListSchemaFiltersValue0(GetListSchemaFiltersValue0(databaseID))
	if err != nil {
		return nil, err
	}

	res, err := cw.GetApiV1DatabaseWithResponse(ctx, &GetApiV1DatabaseParams{
		Q: GetListSchema{
			Filters: []struct {
				Col   string                      `json:"col"`
				Opr   string                      `json:"opr"`
				Value GetListSchema_Filters_Value `json:"value"`
			}{
				{Col: "id", Opr: "eq", Value: v},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("failed to find database, status code: %d, body: %s", res.StatusCode(), string(res.Body))
	}

	if len(res.JSON200.Result) == 0 {
		return nil, fmt.Errorf("database not found: %d", databaseID)
	}

	return &res.JSON200.Result[0], nil

}

// DeleteDatabase deletes the database with the given databaseID.
func (cw *ClientWrapper) DeleteDatabase(ctx context.Context, databaseID int) error {
	reqEditor, err := cw.createCsrfTokenRequestEditor()
	if err != nil {
		return err
	}

	res, err := cw.DeleteApiV1DatabasePk(ctx, databaseID, reqEditor)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to delete database, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

// UpdateDatabase updates the database with the given databaseID using the provided database data.
func (cw *ClientWrapper) UpdateDatabase(ctx context.Context, databaseID int, database DatabaseRestApiPut) error {
	reqEditor, err := cw.createCsrfTokenRequestEditor()
	if err != nil {
		return err
	}

	res, err := cw.PutApiV1DatabasePk(ctx, databaseID, database, reqEditor)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to update database, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}

// ExecuteTestDatabaseConnection tests the database connection with the given connection parameters.
func (cw *ClientWrapper) ExecuteTestDatabaseConnection(ctx context.Context, body DatabaseTestConnectionSchema) error {
	reqEditor, err := cw.createCsrfTokenRequestEditor()
	if err != nil {
		return err
	}

	res, err := cw.PostApiV1DatabaseTestConnection(ctx, body, reqEditor)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		defer func() { res.Body.Close() }()
		msg, err := io.ReadAll(res.Body)

		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("failed to test database connection, status code: %d, body: %s", res.StatusCode, string(msg))
	}
	return nil
}
