package mux

import (
	"context"

	"golang.org/x/exp/slices"

	"github.com/lumavpn/luma/component/auth"
	"github.com/lumavpn/luma/proxy/inbound"
)

type contextKey string

var ctxKeyAdditions = contextKey("Additions")

func WithAdditions(ctx context.Context, additions ...inbound.Option) context.Context {
	return context.WithValue(ctx, ctxKeyAdditions, additions)
}

func getAdditions(ctx context.Context) (additions []inbound.Option) {
	if v := ctx.Value(ctxKeyAdditions); v != nil {
		if a, ok := v.([]inbound.Option); ok {
			additions = a
		}
	}
	if user, ok := auth.UserFromContext[string](ctx); ok {
		additions = slices.Clone(additions)
		additions = append(additions, inbound.WithInUser(user))
	}
	return
}
