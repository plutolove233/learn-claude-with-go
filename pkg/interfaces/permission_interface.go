package interfaces

import (
	"claudego/pkg/permissions"
	"context"
)

type PermissionDecider interface {
	Decide(ctx context.Context, req permissions.Request) permissions.Decision
}
