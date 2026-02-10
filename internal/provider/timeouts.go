// Copyright (c) Hironori Tamakoshi <tmkshrnr@gmail.com>
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	Timeout1Min   = 1 * time.Minute
	Timeout5min   = 5 * time.Minute
	Timeout20min  = 20 * time.Minute
	Timeout60min  = 60 * time.Minute
	Timeout24hour = 24 * time.Hour
)

func SetupTimeoutCreate(ctx context.Context, tov timeouts.Value, defaultTimeout time.Duration) (context.Context, context.CancelFunc) {
	createTimeout, diags := tov.Create(ctx, defaultTimeout)

	if diags.HasError() {
		tflog.Info(ctx, fmt.Sprintf("Failed to get create timeout. Use default timeout: %s", createTimeout))
	}

	return context.WithTimeout(ctx, createTimeout)
}

func SetupTimeoutUpdate(ctx context.Context, tov timeouts.Value, defaultTimeout time.Duration) (context.Context, context.CancelFunc) {
	updateTimeout, diags := tov.Update(ctx, defaultTimeout)

	if diags.HasError() {
		tflog.Info(ctx, fmt.Sprintf("Failed to get update timeout. Use default timeout: %s", updateTimeout))
	}

	return context.WithTimeout(ctx, updateTimeout)
}

func SetupTimeoutDelete(ctx context.Context, tov timeouts.Value, defaultTimeout time.Duration) (context.Context, context.CancelFunc) {
	deleteTimeout, diags := tov.Delete(ctx, defaultTimeout)

	if diags.HasError() {
		tflog.Info(ctx, fmt.Sprintf("Failed to get delete timeout. Use default timeout: %s", deleteTimeout))
	}

	return context.WithTimeout(ctx, deleteTimeout)
}
