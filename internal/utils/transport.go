package utils

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"
)

func GetUid(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	return tr.RequestHeader().Get("X-User-ID")
}
