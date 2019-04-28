package main

import (
	"fmt"
	"github.com/temprory/log"
	"github.com/temprory/net"
	"sync/atomic"
	"time"
)

var (
	addr = "127.0.0.1:8888"

	qpsCall = int64(0)
	qps6666 = int64(0)
	qps8888 = int64(0)
)

type HelloRequest struct {
	Name string
}

type HelloReply struct {
	Message string
}

func on6666(client net.ITcpClient, msg net.IMessage) {
	//log.Info("on6666: %v", string(msg.Body()))
	atomic.AddInt64(&qps6666, 1)
	client.SendMsg(msg)
}

func on8888(client net.ITcpClient, msg net.IMessage) {
	//log.Info("on8888: %v", string(msg.Body()))
	atomic.AddInt64(&qps8888, 1)
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
		for {
			time.Sleep(time.Second)
			fmt.Println("qps:", atomic.SwapInt64(&qpsCall, 0), atomic.SwapInt64(&qps6666, 0), atomic.SwapInt64(&qps8888, 0))
		}
	}()
	// go func() {
	// 	i := 0
	// 	for {
	// 		i++
	// 		//time.Sleep(time.Second)
	// 		if err := pool.Client().SendMsg(net.NewMessage(6666, []byte(fmt.Sprintf("hello_%v", i)))); err != nil {
	// 			return
	// 		}
	// 	}
	// }()
	pool.Client().SendMsg(net.NewMessage(6666, []byte(fmt.Sprintf("hello_%v", 1))))
	pool.Client().SendMsg(net.NewMessage(6666, []byte(fmt.Sprintf("hello_%v", 2))))
	for {
		req := &HelloRequest{Name: "temprory"}
		rsp := &HelloReply{}

		err = pool.Call("Hello", req, rsp, time.Second*3)
		if err != nil {
			log.Error("Hello failed: %v", err)
			return
		}
		atomic.AddInt64(&qpsCall, 1)
		//time.Sleep(time.Second)
	}
}
