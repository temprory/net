package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"net/http"
	_ "net/http/pprof"
	"time"
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Message string
}

func onJsonRpc(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.BindJson(req)
	if err != nil {
		ctx.Error("invalid data")
		log.Error("onJsonRpc failed: %v", err)
		return
	}

	err = ctx.WriteJson(&HelloReply{Message: req.Name})
	if err != nil {
		log.Error("onJsonRpc failed: %v", err)
	}
}

var client net.*TcpClient

func main() {
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	server := net.NewTcpServer("rpc")
	server.HandleRpcMethod("JsonRpc.Hello", onJsonRpc, false)

	server.Serve("localhost:8888", time.Second*5)
}
