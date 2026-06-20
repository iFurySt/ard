package requestid

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const Header = "X-Request-ID"

type contextKey struct{}

func With(ctx context.Context, value string) context.Context {
	value = strings.TrimSpace(value)
	if value == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, value)
}

func Ensure(ctx context.Context) (context.Context, string) {
	if value := From(ctx); value != "" {
		return ctx, value
	}
	value := uuid.NewString()
	return With(ctx, value), value
}

func From(ctx context.Context) string {
	value, _ := ctx.Value(contextKey{}).(string)
	return strings.TrimSpace(value)
}

func SetHeader(header http.Header, ctx context.Context) {
	if value := From(ctx); value != "" {
		header.Set(Header, value)
	}
}
