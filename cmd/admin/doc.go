// Package admin doc
// created at 2022/12/14 by ge
package admin

import (
	"context"
	"go-admin/common/log"
	"runtime"
)

// PackageName is the name of this package
var PackageName = func() string {
	pc, _, _, _ := runtime.Caller(0)
	f := runtime.FuncForPC(pc)
	name := f.Name()
	var dot int
	for i := len(name) - 1; i >= 0; i-- {
		if c := name[i]; c == '/' {
			break
		} else if c == '.' {
			dot = i
		}
	}
	return name[:dot]
}()

var startingCtx = log.NewSpanContext(context.Background(), PackageName, "starting")
