package log

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"github.com/uptrace/uptrace-go/uptrace"
	"os"
	"sync"
	"time"
)

const (
	LevelEnv = "LOG_LEVEL"
)

var (
	defaultLoggerOnce sync.Once
	defaultLogger     *logrus.Logger
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetReportCaller(true)
	if lvl := os.Getenv(LevelEnv); lvl != "" {
		level, err := logrus.ParseLevel(lvl)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "logrus parse level %q: %s\n", lvl, err.Error())
		} else {
			logger.SetLevel(level)
		}
	}
	return logger
}

func Logger() *logrus.Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLogger()
		if UptraceOk() {
			defaultLogger.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
				logrus.WarnLevel,
				logrus.InfoLevel,
				logrus.DebugLevel,
			)))
			logrus.RegisterExitHandler(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := uptrace.Shutdown(ctx); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "shutdown uptrace: %s\n", err.Error())
				}
				_, _ = fmt.Fprintln(os.Stderr, "shutdown uptrace success")
			})
		}
	})
	return defaultLogger
}
