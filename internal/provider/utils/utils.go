// Copyright Hironori Tamakoshi <tmkshrnr@gmail.com> 2026
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func StringsToTlist(values []string) types.List {
	listValue, _ := types.ListValueFrom(context.Background(), types.StringType, values)
	return listValue
}
