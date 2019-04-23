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

func onHello(client net.ITcpClient, msg net.IMessage) {
	log.Info("onHello: %v", string(msg.Body()))
}

func main() {
	poolSize := 3
	engine := net.NewTcpEngine()
	engine.Handle(6666, onHello)
	engine.Handle(8888, onHello)
	pool, err := net.NewRpcClientPool(addr, engine, nil, poolSize, nil)
	if err != nil {
		log.Debug("NewReqClient Error: ", err)
		return
	}

	go func() {
		i := 0
		for {
			i++
			time.Sleep(time.Second)
			if err := pool.Client().SendMsg(net.NewMessage(6666, []byte(fmt.Sprintf("hello_666_%v", i)))); err != nil {
				return
			}
		}
	}()

	for {
		req := &HelloRequest{Name: "temprory"}
		rsp := &HelloReply{}

		err = pool.Call("Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Error("Hello failed: %v", err)
			return
		}
		time.Sleep(time.Second)
	}
}
