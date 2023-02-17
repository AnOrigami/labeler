package log

import (
	"context"
	"fmt"
	"github.com/uptrace/uptrace-go/uptrace"
	"go-admin/common/global"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"os"
	"runtime"
	"sync"
)

const (
	UptraceDsnEnv  = "UPTRACE_DSN"
	ServiceNameEnv = "SERVICE_NAME"
	ServiceEnvEnv  = "SERVICE_ENV"
)

var (
	uptraceOnce sync.Once
	uptraceOk   bool
)

func UptraceOk() bool {
	uptraceOnce.Do(func() {
		dsn, exists := os.LookupEnv(UptraceDsnEnv)
		if !exists {
			return
		}
		uptraceOk = true
		serviceName, _ := os.LookupEnv(ServiceNameEnv)
		if serviceName == "" {
			serviceName = "scrm_unknown"
		}
		opts := []uptrace.Option{
			uptrace.WithDSN(dsn),
			uptrace.WithServiceName(serviceName),
			uptrace.WithServiceVersion(global.Version),
		}
		if env, _ := os.LookupEnv(ServiceEnvEnv); env != "" {
			opts = append(opts, uptrace.WithDeploymentEnvironment(env))
		}
		uptrace.ConfigureOpentelemetry(opts...)
	})
	return uptraceOk
}

func WithTracer(ctx context.Context, moduleName, spanName string, f func(ctx context.Context) error) error {
	if !UptraceOk() {
		return f(ctx)
	}
	ctx, span := otel.Tracer(moduleName).Start(ctx, spanName)
	defer func() {
		if err := recover(); err != nil {
			stackTrace := make([]byte, 10240)
			n := runtime.Stack(stackTrace, false)
			span.SetAttributes(Key("exception.stacktrace").String(string(stackTrace[:n])))
			msg := fmt.Errorf("panic: %v", err)
			span.RecordError(msg)
			span.SetStatus(codes.Error, msg.Error())
		}
		span.End()
	}()
	return f(ctx)
}

func NewSpanContext(ctx context.Context, moduleName, spanName string) context.Context {
	if !UptraceOk() {
		return ctx
	}
	ctx, span := otel.Tracer(moduleName).Start(ctx, spanName)
	span.End()
	return ctx
}

type Key = attribute.Key

func LogAttr(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}
