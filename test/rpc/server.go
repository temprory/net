package main

import (
	//"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"time"
)

var (
	addr = "0.0.0.0:8888"

	CMD_ECHO     = uint32(6666)
	CMD_SVR_CALL = uint32(8888)
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	IMessage string
}

func onEcho(client *net.TcpClient, msg net.IMessage) {
	//log.Info("onEcho: %v, %v", msg.Cmd(), string(msg.Body()))
	client.SendMsg(msg)
}

func onHello(ctx *net.RpcContext) {
	req := &HelloRequest{}
	err := ctx.Bind(req)
	if err != nil {
		log.Error("onHello failed: %v", err)
		return
	}
	err = ctx.Write(&HelloReply{IMessage: req.Name})
	if err != nil {
		log.Error("onHello failed: %v", err)
	}

	// if ctx.Client().UserData() == nil {
	// 	ctx.Client().SetUserData(true)
	// 	go func() {
	// 		i := 0
	// 		for {
	// 			i++
	// 			time.Sleep(time.Second / 10)
	// 			if err := ctx.Client().SendMsg(net.NewMessage(CMD_SVR_CALL, []byte(fmt.Sprintf("hello_%v", i)))); err != nil {
	// 				log.Info("client exit, stop hello loop")
	// 				return
	// 			}
	// 		}
	// 	}()
	// }
}

func main() {
	server := net.NewTcpServer("rpc")
	server.SetSendQueueSize(4096)
	//处理命令号
	server.Handle(CMD_ECHO, onEcho)

	//初始化路由
	server.HandleRpcMethod("Hello", onHello, true)

	server.Serve(addr, time.Second*5)
}
