// Package service doc
// created at 2022/8/30 by ge
package service

import (
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
