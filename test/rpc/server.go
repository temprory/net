package main

import (
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

var (
	addr = "127.0.0.1:8888"
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Message string
}

func onHello(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.Bind(req)
	if err != nil {
		log.Error("onHello failed: %v", err)
		return
	}
	err = ctx.Write(&HelloReply{Message: req.Name})
	if err != nil {
		log.Error("onHello failed: %v", err)
	}
}

func main() {
	server := net.NewTcpServer("rpc")

	//初始化路由
	server.HandleRpcMethod("Hello", onHello, true)

	server.Serve(addr, time.Second*5)
}
