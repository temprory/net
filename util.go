package net

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"runtime"
	"time"
)

var (
	inited = false
)

const (
	maxStack  = 20
	separator = "---------------------------------------\n"
)

func Ping(addr string, to time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, to)
	if err == nil {
		defer conn.Close()

	}
	return err
}

func handlePanic() interface{} {
	if err := recover(); err != nil {
		errstr := fmt.Sprintf("\n%sruntime error: %v\ntraceback:\n", separator, err)

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

		logDebug(errstr)

		return err
	}
	return nil
}

func safe(cb func()) {
	defer handlePanic()
	cb()
}

func safeGo(cb func()) {
	go func() {
		defer handlePanic()
		cb()
	}()
}

func handleSignal(handler func(sig os.Signal)) {
	if !inited {
		inited = true
		chSignal := make(chan os.Signal, 1)
		//signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		signal.Notify(chSignal)
		for {
			if sig, ok := <-chSignal; ok {
				logDebug("Recv Signal: %v", sig)

				if handler != nil {
					handler(sig)
				}
			} else {
				return
			}
		}
	}
}

func gzipCompress(data []byte) []byte {
	var in bytes.Buffer
	w := gzip.NewWriter(&in)
	w.Write(data)
	w.Close()
	return in.Bytes()
}

func gzipUnCompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	r, _ := gzip.NewReader(b)
	defer r.Close()
	undatas, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return undatas, nil
}
