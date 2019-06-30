package rpc

import (
	"fmt"
	log "github.com/kataras/golog"
	"runtime"
)

const (
	maxStack  = 20
	separator = "---------------------------------------\n"
)

func handlePanic() {
	if err := recover(); err != nil {
		errstr := fmt.Sprintf("%sruntime error: %v\ntraceback:\n", separator, err)

		i := 2
		for {
			pc, file, line, ok := runtime.Caller(i)
			if !ok || i > maxStack {
				break
			}
			errstr += fmt.Sprintf("    stack: %d %v [file: %s] [func: %s] [line: %d]\n", i-1, ok, file, runtime.FuncForPC(pc).Name(), line)
			i++
		}
		errstr += separator

		log.Error(errstr)
	}
}

func safe(cb func()) {
	defer handlePanic()
	cb()
}
