package mux

import (
	"context"
	"slices"

	"github.com/lumavpn/luma/component/auth"
	"github.com/lumavpn/luma/proxy/inbound"
)

type contextKey string

var ctxKeyAdditions = contextKey("Additions")

func WithOptions(ctx context.Context, options ...inbound.Option) context.Context {
	return context.WithValue(ctx, ctxKeyAdditions, options)
}

func getOptions(ctx context.Context) (options []inbound.Option) {
	if v := ctx.Value(ctxKeyAdditions); v != nil {
		if a, ok := v.([]inbound.Option); ok {
			options = a
		}
	}
	if user, ok := auth.UserFromContext[string](ctx); ok {
		options = slices.Clone(options)
		options = append(options, inbound.WithInUser(user))
	}
	return
}
