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

func on6666(client net.ITcpClient, msg net.IMessage) {
	log.Info("on6666: %v", string(msg.Body()))
}

func on8888(client net.ITcpClient, msg net.IMessage) {
	log.Info("on8888: %v", string(msg.Body()))
}

func main() {
	poolSize := 5
	engine := net.NewTcpEngine()

	engine.Handle(6666, on6666)
	engine.Handle(8888, on8888)
	pool, err := net.NewRpcClientPool(addr, engine, nil, poolSize, nil)
	if err != nil {
		log.Error("NewReqClient Error: %v", err)
		return
	}

	go func() {
		i := 0
		for {
			i++
			time.Sleep(time.Second)
			if err := pool.Client().SendMsg(net.NewMessage(6666, []byte(fmt.Sprintf("hello_%v", i)))); err != nil {
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
