package rpc

import "context"

type onewayKey struct{}

func CtxWithOneWay(ctx context.Context) context.Context {
	return context.WithValue(ctx, onewayKey{}, true)
}

func IsOneWay(ctx context.Context) bool {
	val := ctx.Value(onewayKey{})
	oneway, ok := val.(bool)
	return ok && oneway
}
