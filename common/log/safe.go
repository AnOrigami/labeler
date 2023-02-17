package log

import (
	"github.com/sirupsen/logrus"
	"os"
)

type SafeGoConfig struct {
	Name        string
	PanicToExit bool
}

type SafeGoOption func(opt *SafeGoConfig)

func PanicToExit() SafeGoOption {
	return func(opt *SafeGoConfig) {
		opt.PanicToExit = true
	}
}

func WithName(name string) SafeGoOption {
	return func(opt *SafeGoConfig) {
		opt.Name = name
	}
}

func Exit(code int) {
	os.Exit(code)
}

func SafeGo(f func(), opts ...SafeGoOption) {
	config := &SafeGoConfig{}
	for _, opt := range opts {
		opt(config)
	}
	go func() {
		defer func() {
			recovered := recover()
			if recovered != nil {
				level := logrus.ErrorLevel
				if config.PanicToExit {
					level = logrus.FatalLevel
				}
				Logger().Log(level, recovered)
			}
		}()
		f()
	}()
}
