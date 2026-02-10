// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package client

import "time"

type SupersetUser struct {
	Active           bool
	ChangedBy        *SupersetUser
	ChangedOn        time.Time
	CreatedBy        *SupersetUser
	CreatedOn        time.Time
	Email            string
	FirstName        string
	LastName         string
	Id               int
	Groups           []SupersetGroup
	LastLogin        time.Time
	LoginCount       int
	FailedLoginCount int
	Roles            []SupersetRole
	Username         string
}

type SupersetGroup struct {
	Description string
	Id          int
	Label       string
	Name        string
	Roles       []SupersetRole
	Users       []SupersetUser
}

type SupersetRole struct {
	Id   int
	Name string
}

type SupersetPermission struct {
	Id             int
	PermissionName string
	ViewMenuName   string
}

type RolePermission struct {
	Id             int
	PermissionName string
	ViewMenuName   string
}

type SupersetDatabase struct {
	AllowCtas                 bool
	AllowCvas                 bool
	AllowsDml                 bool
	AllowsVirtualTable        bool
	AllowFileUpload           bool
	AllowMultiCatalog         bool
	AllowRunAsync             bool
	AllowsCostEstimate        bool
	AllowsSubquery            bool
	AllowsVirtualTableExplore bool
	DatabaseName              string
	ChangedBy                 *SupersetUser
	ChangedOn                 time.Time
	CreatedBy                 *SupersetUser
}
