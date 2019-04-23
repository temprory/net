package main

import (
	"fmt"
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

func on666(client net.ITcpClient, msg net.IMessage) {
	log.Info("on666: %v", string(msg.Body()))
	client.SendMsg(msg)
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

	if ctx.Client().UserData() == nil {
		ctx.Client().SetUserData(true)
		go func() {
			i := 0
			for {
				i++
				time.Sleep(time.Second)
				if err := ctx.Client().SendMsg(net.NewMessage(8888, []byte(fmt.Sprintf("hello_%v", i)))); err != nil {
					log.Info("client exit, stop hello loop")
					return
				}
			}
		}()
	}
}

func main() {
	server := net.NewTcpServer("rpc")

	//处理命令号
	server.Handle(6666, on666)

	//初始化路由
	server.HandleRpcMethod("Hello", onHello, true)

	server.Serve(addr, time.Second*5)
}
