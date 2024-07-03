package service

import (
	"context"

	"github.com/lumavpn/luma/util"
)

func ContextWithRegistry(ctx context.Context, registry Registry) context.Context {
	return context.WithValue(ctx, util.DefaultValue[*Registry](), registry)
}

func ContextWithDefaultRegistry(ctx context.Context) context.Context {
	if RegistryFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, util.DefaultValue[*Registry](), NewRegistry())
}

func RegistryFromContext(ctx context.Context) Registry {
	registry := ctx.Value(util.DefaultValue[*Registry]())
	if registry == nil {
		return nil
	}
	return registry.(Registry)
}

func FromContext[T any](ctx context.Context) T {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return util.DefaultValue[T]()
	}
	service := registry.Get(util.DefaultValue[*T]())
	if service == nil {
		return util.DefaultValue[T]()
	}
	return service.(T)
}

func PtrFromContext[T any](ctx context.Context) *T {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		return nil
	}
	servicePtr := registry.Get(util.DefaultValue[*T]())
	if servicePtr == nil {
		return nil
	}
	return servicePtr.(*T)
}

func ContextWith[T any](ctx context.Context, service T) context.Context {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		registry = NewRegistry()
		ctx = ContextWithRegistry(ctx, registry)
	}
	registry.Register(util.DefaultValue[*T](), service)
	return ctx
}

func ContextWithPtr[T any](ctx context.Context, servicePtr *T) context.Context {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		registry = NewRegistry()
		ctx = ContextWithRegistry(ctx, registry)
	}
	registry.Register(util.DefaultValue[*T](), servicePtr)
	return ctx
}

func MustRegister[T any](ctx context.Context, service T) {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		panic("missing service registry in context")
	}
	registry.Register(util.DefaultValue[*T](), service)
}

func MustRegisterPtr[T any](ctx context.Context, servicePtr *T) {
	registry := RegistryFromContext(ctx)
	if registry == nil {
		panic("missing service registry in context")
	}
	registry.Register(util.DefaultValue[*T](), servicePtr)
}
