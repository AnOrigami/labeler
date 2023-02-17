package log

import "context"

type noCancelCtx struct {
	context.Context
}

func (ctx *noCancelCtx) Done() <-chan struct{} {
	return nil
}

func WithNoCancel(ctx context.Context) context.Context {
	return &noCancelCtx{Context: ctx}
}
