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

func onHello(client net.ITcpClient, msg net.IMessage) {
	log.Info("onHello: %v", string(msg.Body()))
}

func main() {
	//log.DefaultLogger.FullPath = false
	//client, err := net.NewRpcClient(addr, nil, nil, nil)
	//log.DefaultLogger.AddFileIgnorePath("github.com")
	poolSize := 10
	engine := net.NewTcpEngine()
	engine.Handle(8888, onHello)
	client, err := net.NewRpcClientPool(addr, engine, nil, poolSize, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
		return
	}

	for {
		req := &HelloRequest{Name: "temprory"}
		rsp := &HelloReply{}

		err = client.Call("Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Error("Hello failed: %v", err)
			return
		}
		time.Sleep(time.Second)
	}
}
